package merge

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/pterm/pterm"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/cmd/media/subtitle/internal/subcleaner"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/subtitle/merge/internal/mkvmerge"
	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/media"
	"github.com/jeremiergz/nas-cli/internal/prompt"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/service/str"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	mergeDesc         = "Merge tracks using MKVMerge tool"
	cleanFirst        bool
	delete            bool
	dryRun            bool
	maxParallel       int
	overrideLanguage  bool
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
		Args:    cobra.MaximumNArgs(1),
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

			selectedDir := "."
			if len(args) > 0 {
				selectedDir = args[0]
			}

			return fsutil.InitializeWorkingDir(selectedDir)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			files, err := media.Files(config.WD, videoExtensions, false)
			if err != nil {
				return err
			}

			if len(files) == 0 {
				pterm.Success.Println("Nothing to process")
				return nil
			}

			print(out, files)
			if dryRun {
				return nil
			}

			fmt.Fprintln(out)

			var p prompt.Prompter
			if yes {
				p = prompt.NewAuto()
			} else {
				p = prompt.NewInteractive()
			}

			shouldProcess, err := p.Confirm(
				fmt.Sprintf("Process %d file(s)?", len(files)),
				true,
			)
			if err != nil {
				return nil
			}
			if !shouldProcess {
				return nil
			}

			fmt.Fprintln(out)

			err = process(cmd.Context(), out, files, !delete)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&cleanFirst, "clean", "c", false, "clean subtitle files before merging")
	cmd.Flags().BoolVarP(&delete, "delete", "d", false, "delete original files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print result without processing it")
	cmd.Flags().IntVarP(&maxParallel, "max-parallel", "p", 0, "maximum number of parallel processes. 0 means no limit")
	cmd.Flags().BoolVar(&overrideLanguage, "override-language", false, "replace existing subtitle tracks with incoming ones of the same language")
	cmd.Flags().StringArrayVarP(&subtitleLanguages, "language", "l", []string{"eng", "fre"}, "language tracks to merge")
	cmd.Flags().StringVar(&subtitleExtension, "sub-ext", util.AcceptedSubtitleExtension, "filter subtitles by extension")
	cmd.Flags().StringArrayVarP(&videoExtensions, "video-ext", "e", util.AcceptedVideoExtensions, "filter video files by extension")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "automatic yes to prompts")

	return cmd
}

// Prints given files and their subtitles as a tree.
func print(w io.Writer, files []*media.File) {
	lw := cmdutil.NewListWriter()
	filesCount := len(files)

	lw.AppendItem(
		fmt.Sprintf(
			"%s (%d %s)",
			config.WD,
			filesCount,
			lo.Ternary(filesCount <= 1, "file", "files"),
		),
	)

	lw.Indent()
	for _, file := range files {
		lw.AppendItem(file.Name())

		lw.Indent()
		for lang, subtitle := range file.Subtitles() {
			flag := util.ToLanguageFlag(lang)
			var str string
			if flag != "" {
				str = fmt.Sprintf("%s  %s", flag, subtitle)
			} else {
				langCode := lang[0:1] + lang[1:2]
				str = fmt.Sprintf("%s  %s", strings.ToUpper(langCode), subtitle)
			}
			lw.AppendItem(str)
		}
		lw.UnIndent()
	}

	fmt.Fprintln(w, lw.Render())
}

// Merges language tracks into one video file.
// If --clean is set, each file's associated subtitle files are cleaned just before they are merged.
func process(ctx context.Context, w io.Writer, files []*media.File, keepOriginal bool) error {
	// Pre-count trackers: one merge tracker per file + one clean tracker per subtitle (if cleanFirst).
	totalTrackers := len(files)
	allFileNames := lo.Map(files, func(file *media.File, _ int) string { return file.Basename() })
	if cleanFirst {
		for _, file := range files {
			subtitleNames := lo.Values(file.Subtitles())
			totalTrackers += len(subtitleNames)
			allFileNames = append(allFileNames, subtitleNames...)
		}
	}

	pw := cmdutil.NewProgressWriter(w, totalTrackers)

	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(cmdutil.MaxConcurrentGoroutines)
	if maxParallel > 0 {
		eg.SetLimit(maxParallel)
	}

	padder := str.NewPadder(allFileNames)

	for _, file := range files {
		// Build per-file cleaners if --clean is set.
		var cleaners []svc.Runnable
		if cleanFirst {
			fileSubtitles := file.Subtitles()
			subtitleNames := lo.Values(fileSubtitles)
			if len(subtitleNames) > 0 {
				for _, subtitleName := range subtitleNames {
					paddingLength := padder.PaddingLength(subtitleName, 1)
					tracker := &progress.Tracker{
						DeferStart: true,
						Message:    fmt.Sprintf("%s%*s", subtitleName, paddingLength, " "),
						Total:      100,
					}
					pw.AppendTracker(tracker)
					subtitlePath := filepath.Join(filepath.Dir(file.FilePath()), subtitleName)
					cleaner := subcleaner.
						New(subtitlePath, true).
						SetOutput(w).
						SetTracker(tracker)
					cleaners = append(cleaners, cleaner)
				}
			}
		}

		// Merge tracker.
		paddingLength := padder.PaddingLength(file.Basename(), 1)
		mergeTracker := &progress.Tracker{
			DeferStart: true,
			Message:    fmt.Sprintf("%s%*s", file.Basename(), paddingLength, " "),
			Total:      100,
		}
		pw.AppendTracker(mergeTracker)
		merger := mkvmerge.
			New(file, keepOriginal, overrideLanguage).
			SetOutput(w).
			SetTracker(mergeTracker)

		eg.Go(func() error {
			// Clean each subtitle sequentially, then merge.
			for _, cleaner := range cleaners {
				if err := cleaner.Run(ctx); err != nil {
					return err
				}
			}
			return merger.Run(ctx)
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
