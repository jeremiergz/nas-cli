package merge

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
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

			files, err := model.Files(config.WD, videoExtensions, false)
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

			fmt.Fprintln(out)

			filesToProcess := []*model.File{}
			for _, file := range files {
				if !yes {
					shouldProcess := svc.Console.AskConfirmation(
						fmt.Sprintf("Process %q?", file.FullName()),
						true,
					)
					if !shouldProcess {
						continue
					}
				}
				filesToProcess = append(filesToProcess, file)
			}

			if len(filesToProcess) == 0 {
				return nil
			}

			fmt.Fprintln(out)

			err = process(cmd.Context(), out, filesToProcess, !delete)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&delete, "delete", "d", false, "delete original files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print result without processing it")
	cmd.Flags().IntVarP(&maxParallel, "max-parallel", "p", 0, "maximum number of parallel processes. 0 means no limit")
	cmd.Flags().StringArrayVarP(&subtitleLanguages, "language", "l", []string{"eng", "fre"}, "language tracks to merge")
	cmd.Flags().StringVar(&subtitleExtension, "sub-ext", util.AcceptedSubtitleExtension, "filter subtitles by extension")
	cmd.Flags().StringArrayVarP(&videoExtensions, "video-ext", "e", util.AcceptedVideoExtensions, "filter video files by extension")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "automatic yes to prompts")

	return cmd
}

// Prints given files and their subtitles as a tree.
func print(w io.Writer, files []*model.File) {
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
func process(ctx context.Context, w io.Writer, files []*model.File, keepOriginal bool) error {
	pw := cmdutil.NewProgressWriter(w, len(files))

	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(cmdutil.MaxConcurrentGoroutines)
	if maxParallel > 0 {
		eg.SetLimit(maxParallel)
	}

	padder := str.NewPadder(lo.Map(files, func(file *model.File, _ int) string { return file.Basename() }))

	mergers := make([]svc.Runnable, len(files))
	for index, file := range files {
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
		mergers[index] = merger
	}
	for _, merger := range mergers {
		eg.Go(func() error {
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
