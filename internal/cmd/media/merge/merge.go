package merge

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/disiqueira/gotree/v3"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/manifoldco/promptui"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/cmd/media/merge/internal/mkvmerge"
	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/service/str"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

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
			if cmdutil.DebugMode {
				fmt.Fprintf(cmd.OutOrStdout(), "%s PersistentPreRunE\n", cmd.CommandPath())
			}

			err := cmdutil.CallParentPersistentPreRunE(cmd, args)
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

	padder := str.NewPadder(lo.Map(files, func(file *model.File, _ int) string { return file.Basename() }))

	wg := sync.WaitGroup{}
	wg.Add(1)
	for _, file := range files {
		paddingLength := padder.PaddingLength(file.Basename(), 1)
		tracker := &progress.Tracker{
			DeferStart: true,
			Message:    fmt.Sprintf("%s%*s", file.Basename(), paddingLength, " "),
			Total:      100,
		}
		pw.AppendTracker(tracker)
		merger := mkvmerge.
			New(file, keepOriginal).
			SetOutput(w).
			SetTracker(tracker)

		eg.Go(func() error {
			wg.Wait()
			err := merger.Run()
			if err != nil {
				return err
			}
			return nil
		})
	}
	wg.Done()
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
