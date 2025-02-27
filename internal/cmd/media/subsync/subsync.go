package subsync

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/cmd/media/subsync/internal/subsync"
	"github.com/jeremiergz/nas-cli/internal/config"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/service/str"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	subsyncDesc       = "Synchronize subtitle using SubSync tool"
	dryRun            bool
	maxParallel       int
	streamLang        string
	streamType        string
	subtitleExtension string
	subtitleLang      string
	videoExtensions   []string
	videoLang         string
	yes               bool
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "subsync <directory>",
		Aliases: []string{"sub"},
		Short:   subsyncDesc,
		Long:    subsyncDesc + ".",
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

			subtitleFiles := fsutil.List(config.WD, []string{subtitleExtension}, nil, false)
			if len(subtitleFiles) == 0 {
				svc.Console.Success("No subtitle file to process")
				return nil
			}

			videoFiles := fsutil.List(config.WD, videoExtensions, nil, false)
			if len(videoFiles) == 0 {
				svc.Console.Success("No video file to process")
				return nil
			}

			sort.Sort(util.SortAlphabetic(videoFiles))
			sort.Sort(util.SortAlphabetic(subtitleFiles))

			displayList(out, config.WD, videoFiles, subtitleFiles)
			if dryRun {
				return nil
			}

			if !yes {
				fmt.Fprintln(out)
				shouldProcess := svc.Console.AskConfirmation("Process?", true)
				if !shouldProcess {
					return nil
				}
			}

			fmt.Fprintln(out)

			err := process(cmd.Context(), out, videoFiles, subtitleFiles)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print result without processing it")
	cmd.Flags().IntVarP(&maxParallel, "max-parallel", "p", 0, "maximum number of parallel processes. 0 means no limit")
	cmd.Flags().StringVar(&streamLang, "stream", "eng", "stream ISO 639-3 language code")
	cmd.Flags().StringVar(&streamType, "stream-type", "", "stream type (audio|sub)")
	cmd.Flags().StringVar(&subtitleExtension, "sub-ext", util.AcceptedSubtitleExtension, "filter subtitles by extension")
	cmd.Flags().StringVar(&subtitleLang, "sub-lang", "eng", "subtitle ISO 639-3 language code")
	cmd.Flags().StringArrayVarP(&videoExtensions, "video-ext", "e", util.AcceptedVideoExtensions, "filter video files by extension")
	cmd.Flags().StringVar(&videoLang, "video-lang", "eng", "video ISO 639-3 language code")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "automatic yes to prompts")

	return cmd
}

func displayList(out io.Writer, wd string, videos []string, subtitles []string) {
	lw := cmdutil.NewListWriter()
	lw.AppendItem(wd)
	for index, video := range videos {
		lw.Indent()
		lw.AppendItem(index + 1)

		lw.Indent()
		lw.AppendItem(subtitles[index])
		lw.AppendItem(video)

		lw.UnIndentAll()
	}
	fmt.Fprintln(out, lw.Render())
}

func process(ctx context.Context, out io.Writer, videoFiles, subtitleFiles []string) error {
	pw := cmdutil.NewProgressWriter(out, len(videoFiles))

	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(cmdutil.MaxConcurrentGoroutines)
	if maxParallel > 0 {
		eg.SetLimit(maxParallel)
	}

	padder := str.NewPadder(videoFiles)

	syncers := make([]svc.Runnable, len(videoFiles))
	for index, videoFile := range videoFiles {
		paddingLength := padder.PaddingLength(videoFile, 12)
		tracker := &progress.Tracker{
			DeferStart: true,
			Message:    fmt.Sprintf("%s%*s", videoFile, paddingLength, " "),
			Total:      100,
		}
		pw.AppendTracker(tracker)
		syncer := subsync.
			New(
				videoFile,
				videoLang,
				subtitleFiles[index],
				subtitleLang,
				streamLang,
				streamType,
				strings.Replace(videoFile, path.Ext(videoFile), fmt.Sprintf(".%s.srt", subtitleLang), 1),
			).
			SetOutput(out).
			SetTracker(tracker)
		syncers[index] = syncer
	}
	for _, syncer := range syncers {
		eg.Go(func() error {
			return syncer.Run(ctx)
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
