package subclean

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/cmd/media/subclean/internal/subcleaner"
	"github.com/jeremiergz/nas-cli/internal/config"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/service/str"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	subcleanDesc = "Clean subtitle files"
	delete       bool
	dryRun       bool
	yes          bool
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "subclean <directory>",
		Aliases: []string{"subc"},
		Short:   subcleanDesc,
		Long:    subcleanDesc + ".",
		Args:    cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmdutil.DebugMode {
				fmt.Fprintf(cmd.OutOrStdout(), "%s PreRunE\n", cmd.CommandPath())
			}

			_, err := exec.LookPath(cmdutil.CommandSubsync)
			if err != nil {
				return fmt.Errorf("command not found: %s", cmdutil.CommandSubsync)
			}

			selectedDir := "."
			if len(args) > 0 {
				selectedDir = args[0]
			}

			err = fsutil.InitializeWorkingDir(selectedDir)
			if err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			subtitleFiles := fsutil.List(config.WD, []string{"srt"}, nil, true)

			print(out, subtitleFiles)
			if dryRun {
				return nil
			}

			fmt.Fprintln(out)

			if !yes {
				shouldProcess := svc.Console.AskConfirmation(
					fmt.Sprintf("Process %d file(s)?", len(subtitleFiles)),
					true,
				)
				if !shouldProcess {
					return nil
				}
			}

			fmt.Fprintln(out)

			err := process(cmd.Context(), out, subtitleFiles, !delete)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&delete, "delete", "d", false, "delete original files")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print result without processing it")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "automatic yes to prompts")

	return cmd
}

// Prints given subtitles as a tree.
func print(w io.Writer, files []string) {
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
		lw.AppendItem(file)
	}

	fmt.Fprintln(w, lw.Render())
}

// Cleans subtitle files by applying the following transformations:
//   - Merge duplicate timestamps
func process(ctx context.Context, w io.Writer, subtitleFiles []string, keepOriginal bool) error {
	pw := cmdutil.NewProgressWriter(w, len(subtitleFiles))

	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(cmdutil.MaxConcurrentGoroutines)

	padder := str.NewPadder(subtitleFiles)

	cleaners := make([]svc.Runnable, len(subtitleFiles))
	for index, subtitleFile := range subtitleFiles {
		subtitleFilePath := filepath.Join(config.WD, subtitleFile)
		paddingLength := padder.PaddingLength(subtitleFile, 1)
		tracker := &progress.Tracker{
			DeferStart: true,
			Message:    fmt.Sprintf("%s%*s", subtitleFile, paddingLength, " "),
			Total:      100,
		}
		pw.AppendTracker(tracker)
		cleaner := subcleaner.New(subtitleFilePath, keepOriginal).
			SetOutput(w).
			SetTracker(tracker)
		cleaners[index] = cleaner
	}
	for _, cleaner := range cleaners {
		eg.Go(func() error {
			return cleaner.Run(ctx)
		})
	}
	if err := eg.Wait(); err != nil {
		return fmt.Errorf("failed to clean subtitle files: %w", err)
	}

	for pw.IsRenderInProgress() {
		if pw.LengthActive() == 0 {
			pw.Stop()
		}
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}
