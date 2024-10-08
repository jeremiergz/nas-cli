package merge

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/disiqueira/gotree/v3"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/manifoldco/promptui"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

type backup struct {
	currentPath  string
	originalPath string
}

var (
	mergeDesc         = "Merge tracks using MKVMerge tool"
	delete            bool
	dryRun            bool
	maxParallel       int
	subtitleExtension string
	subtitleLanguages []string
	videoExtensions   []string
	yes               bool
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "merge <directory>",
		Aliases: []string{"mrg"},
		Short:   mergeDesc,
		Long:    mergeDesc + ".",
		Args:    cobra.MinimumNArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := cmdutil.CallParentPersistentPreRunE(cmd.Parent(), args)
			if err != nil {
				return err
			}

			_, err = exec.LookPath(cmdutil.CommandMKVMerge)
			if err != nil {
				return fmt.Errorf("command not found: %s", cmdutil.CommandMKVMerge)
			}

			return fsutil.InitializeWorkingDir(args[0])
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			files, err := model.Files(config.WD, videoExtensions)
			if err != nil {
				return err
			}

			if len(files) == 0 {
				svc.Console.Success("Nothing to process")
				return nil
			}

			print(out, files)
			if dryRun {
				return nil
			}

			if !yes {
				fmt.Fprintln(out)
				prompt := promptui.Prompt{
					Label:     "Process",
					IsConfirm: true,
					Default:   "y",
				}
				input, err := prompt.Run()
				if err != nil {
					if err.Error() == "^C" || input != "" {
						return nil
					}
					return err
				}
			}

			fmt.Fprintln(out)

			err = process(cmd.Context(), out, files, !delete)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.PersistentFlags().BoolVarP(&delete, "delete", "d", false, "delete original files")
	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print result without processing it")
	cmd.PersistentFlags().IntVarP(&maxParallel, "max-parallel", "p", 0, "maximum number of parallel processes. 0 means no limit")
	cmd.PersistentFlags().StringArrayVarP(&subtitleLanguages, "language", "l", []string{"eng", "fre"}, "language tracks to merge")
	cmd.PersistentFlags().StringVar(&subtitleExtension, "sub-ext", util.AcceptedSubtitleExtension, "filter subtitles by extension")
	cmd.PersistentFlags().StringArrayVarP(&videoExtensions, "video-ext", "e", util.AcceptedVideoExtensions, "filter video files by extension")
	cmd.PersistentFlags().BoolVarP(&yes, "yes", "y", false, "automatic yes to prompts")

	return cmd
}

// Prints given files and their subtitles as a tree.
func print(w io.Writer, files []*model.File) {
	rootTree := gotree.New(config.WD)
	for _, file := range files {
		fileTree := rootTree.Add(file.Name())
		for lang, subtitle := range file.Subtitles() {
			flag := util.ToLanguageFlag(lang)
			if flag != "" {
				fileTree.Add(fmt.Sprintf("%s  %s", flag, subtitle))
			} else {
				langCode := lang[0:1] + lang[1:2]
				fileTree.Add(fmt.Sprintf("%s  %s", strings.ToUpper(langCode), subtitle))
			}
		}
	}
	toPrint := rootTree.Print()
	lastSpaceRegexp := regexp.MustCompile(`\s$`)
	toPrint = lastSpaceRegexp.ReplaceAllString(toPrint, "")
	fmt.Fprintln(w, toPrint)
}

// Merges language tracks into one video file.
func process(ctx context.Context, w io.Writer, files []*model.File, keepOriginal bool) error {
	pw := cmdutil.NewProgressWriter(w, len(files))

	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(cmdutil.MaxConcurrentGoroutines)
	if maxParallel > 0 {
		eg.SetLimit(maxParallel)
	}

	maxFilenameLength := len(lo.MaxBy(files, func(a, b *model.File) bool {
		return len(a.Basename()) > len(b.Basename())
	}).Basename())

	trackerIndexedByFile := make(map[string]*progress.Tracker, len(files))
	for _, file := range files {
		paddingLength := maxFilenameLength - len(file.Basename())
		if paddingLength > 0 {
			paddingLength += 1
		}
		tracker := &progress.Tracker{
			DeferStart: true,
			Message:    fmt.Sprintf("%s%*s", file.Basename(), paddingLength, " "),
			Total:      100,
		}
		pw.AppendTracker(tracker)
		trackerIndexedByFile[file.Basename()] = tracker
	}

	for _, file := range files {
		eg.Go(func() error {
			tracker := trackerIndexedByFile[file.Basename()]
			subtitles := file.Subtitles()

			// Nothing to do if there are no subtitles.
			if len(subtitles) == 0 {
				tracker.MarkAsDone()
				return nil
			}

			err := merge(tracker, w, file, keepOriginal)
			if err != nil {
				tracker.MarkAsErrored()
				return err
			}

			tracker.MarkAsDone()
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	for pw.IsRenderInProgress() {
		if pw.LengthActive() == 0 {
			pw.Stop()
		}
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func merge(tracker *progress.Tracker, w io.Writer, file *model.File, keepOriginal bool) error {
	videoFileBackupPath := path.Join(config.WD, fmt.Sprintf("%s%s%s", "_", file.Basename(), ".bak"))

	err := os.Rename(file.FilePath(), videoFileBackupPath)
	if err != nil {
		return fmt.Errorf("failed to rename video file: %w", err)
	}

	backups := []backup{
		{currentPath: videoFileBackupPath, originalPath: file.FilePath()},
	}

	options := []string{
		"--gui-mode",
		"--output",
		file.FilePath(),
	}
	for lang, subtitleFile := range file.Subtitles() {
		subtitleFilePath := path.Join(config.WD, subtitleFile)
		subtitleFileBackupPath := path.Join(config.WD, fmt.Sprintf("%s%s%s", "_", subtitleFile, ".bak"))
		os.Rename(subtitleFilePath, subtitleFileBackupPath)
		backups = append(backups, backup{currentPath: subtitleFileBackupPath, originalPath: subtitleFilePath})
		options = append(options, "--language", fmt.Sprintf("0:%s", lang), subtitleFileBackupPath)
	}
	options = append(options, videoFileBackupPath)

	var buf bytes.Buffer

	merge := exec.Command(cmdutil.CommandMKVMerge, options...)
	merge.Stdout = &buf
	merge.Stderr = w

	if err = merge.Start(); err != nil {
		return err
	}

	go func() {
		for !tracker.IsDone() {
			progress, err := getProgress(buf.String())
			if err == nil {
				tracker.SetValue(int64(progress))
			}
			buf.Reset()
			time.Sleep(100 * time.Millisecond)
		}
	}()

	if err = merge.Wait(); err != nil {
		wg := sync.WaitGroup{}
		for _, backupFile := range backups {
			wg.Add(1)
			go func(b backup) {
				defer wg.Done()
				os.Rename(b.currentPath, b.originalPath)
			}(backupFile)
		}
		wg.Wait()
		return err
	}

	os.Chown(file.FilePath(), config.UID, config.GID)
	os.Chmod(file.FilePath(), config.FileMode)

	if !keepOriginal {
		wg := sync.WaitGroup{}
		for _, backupFile := range backups {
			wg.Add(1)
			go func(b backup) {
				defer wg.Done()
				os.Remove(b.currentPath)
			}(backupFile)
		}
		wg.Wait()
	}

	return nil
}

var progressRegexp = regexp.MustCompile(`(?m)(?:progress\s+)(?P<Percentage>\d+)(?:%)`)

func getProgress(str string) (percentage int, err error) {
	allProgressMatches := progressRegexp.FindAllStringSubmatch(str, -1)
	if len(allProgressMatches) == 0 {
		return 0, fmt.Errorf("could not find progress percentage")
	}

	progressMatches := allProgressMatches[len(allProgressMatches)-1]

	if len(progressMatches) != 2 {
		return 0, fmt.Errorf("could not find progress percentage")
	}

	percentageIndex := progressRegexp.SubexpIndex("Percentage")
	if percentageIndex == -1 {
		return 0, fmt.Errorf("could not determine progress percentage")
	}
	percentage, err = strconv.Atoi(progressMatches[percentageIndex])
	if err != nil {
		return 0, fmt.Errorf("could not parse progress percentage: %w", err)
	}

	return percentage, nil
}
