package subsync

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/cmd/media/subsync/internal"
	"github.com/jeremiergz/nas-cli/internal/config"
	svc "github.com/jeremiergz/nas-cli/internal/service"
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
		Args:    cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			_, err := exec.LookPath(cmdutil.CommandSubsync)
			if err != nil {
				return fmt.Errorf("command not found: %s", cmdutil.CommandSubsync)
			}

			err = fsutil.InitializeWorkingDir(args[0])
			if err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			subtitleFiles := fsutil.List(config.WD, []string{subtitleExtension}, nil)
			if len(subtitleFiles) == 0 {
				svc.Console.Success("No subtitle file to process")
				return nil
			}

			videoFiles := fsutil.List(config.WD, videoExtensions, nil)
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

			err := process(cmd.Context(), out, videoFiles, subtitleFiles)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print result without processing it")
	cmd.PersistentFlags().IntVarP(&maxParallel, "max-parallel", "p", 0, "maximum number of parallel processes. 0 means no limit")
	cmd.PersistentFlags().StringVar(&streamLang, "stream", "eng", "stream ISO 639-3 language code")
	cmd.PersistentFlags().StringVar(&streamType, "stream-type", "", "stream type (audio|sub)")
	cmd.PersistentFlags().StringVar(&subtitleExtension, "sub-ext", util.AcceptedSubtitleExtension, "filter subtitles by extension")
	cmd.PersistentFlags().StringVar(&subtitleLang, "sub-lang", "eng", "subtitle ISO 639-3 language code")
	cmd.PersistentFlags().StringArrayVarP(&videoExtensions, "video-ext", "e", util.AcceptedVideoExtensions, "filter video files by extension")
	cmd.PersistentFlags().StringVar(&videoLang, "video-lang", "eng", "video ISO 639-3 language code")
	cmd.PersistentFlags().BoolVarP(&yes, "yes", "y", false, "automatic yes to prompts")

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

	trackerIndexedByVideoFile := make(map[string]*progress.Tracker, len(videoFiles))
	for _, videoFile := range videoFiles {
		tracker := &progress.Tracker{
			DeferStart: true,
			Message:    fmt.Sprintf("%s%11s", videoFile, ""),
			Total:      100,
		}
		pw.AppendTracker(tracker)
		trackerIndexedByVideoFile[videoFile] = tracker
	}

	for index, videoFile := range videoFiles {
		eg.Go(func() error {
			tracker := trackerIndexedByVideoFile[videoFile]
			videoFileExtension := path.Ext(videoFile)
			outFile := strings.Replace(videoFile, videoFileExtension, fmt.Sprintf(".%s.srt", subtitleLang), 1)
			subtitleFile := subtitleFiles[index]
			err := internal.Synchronize(
				tracker,
				videoFile,
				videoLang,
				subtitleFile,
				subtitleLang,
				streamLang,
				streamType,
				outFile,
			)
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

	return nil
}
