package clean

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/manifoldco/promptui"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/service/str"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	cleanDesc         = "Clean tracks using MKVPropEdit tool"
	dryRun            bool
	subtitleExtension string
	subtitleLanguages []string
	videoExtensions   []string
	yes               bool
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clean <directory>",
		Aliases: []string{"cln"},
		Short:   cleanDesc,
		Long:    cleanDesc + ".",
		Args:    cobra.MinimumNArgs(1),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmdutil.DebugMode {
				fmt.Fprintf(cmd.OutOrStdout(), "%s PersistentPreRunE\n", cmd.CommandPath())
			}

			err := cmdutil.CallParentPersistentPreRunE(cmd, args)
			if err != nil {
				return err
			}

			_, err = exec.LookPath(cmdutil.CommandMKVPropEdit)
			if err != nil {
				return fmt.Errorf("command not found: %s", cmdutil.CommandMKVPropEdit)
			}

			err = fsutil.InitializeWorkingDir(args[0])
			if err != nil {
				return err
			}

			return nil
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

			svc.Console.PrintFiles(config.WD, files)
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

			err = process(cmd.Context(), out, files)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print result without processing it")
	cmd.Flags().StringArrayVarP(&subtitleLanguages, "language", "l", []string{"eng", "fre"}, "language tracks to merge")
	cmd.Flags().StringVar(&subtitleExtension, "sub-ext", util.AcceptedSubtitleExtension, "filter subtitles by extension")
	cmd.Flags().StringArrayVarP(&videoExtensions, "video-ext", "e", util.AcceptedVideoExtensions, "filter video files by extension")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "automatic yes to prompts")

	return cmd
}

// Merges show language tracks into one video file.
func process(ctx context.Context, w io.Writer, files []*model.File) error {
	pw := cmdutil.NewProgressWriter(w, len(files))
	pw.Style().Visibility.Tracker = false
	pw.Style().Options.PercentIndeterminate = "   "

	eg, _ := errgroup.WithContext(ctx)

	padder := str.NewPadder(lo.Map(files, func(file *model.File, _ int) string { return file.Basename() }))

	trackerIndexedByFile := make(map[uuid.UUID]*progress.Tracker, len(files))
	for _, file := range files {
		paddingLength := padder.PaddingLength(file.Basename(), 1)
		tracker := &progress.Tracker{
			DeferStart: true,
			Message:    fmt.Sprintf("%s%*s", file.Basename(), paddingLength, " "),
		}
		pw.AppendTracker(tracker)
		trackerIndexedByFile[file.ID()] = tracker
	}

	for _, file := range files {
		tracker := trackerIndexedByFile[file.ID()]
		eg.Go(func() error {
			tracker.Start()

			err := file.Clean()
			if err != nil {
				tracker.MarkAsErrored()
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
