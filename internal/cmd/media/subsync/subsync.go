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
	consolesvc "github.com/jeremiergz/nas-cli/internal/service/console"
	mediasvc "github.com/jeremiergz/nas-cli/internal/service/media"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/ctxutil"
)

var (
	subsyncDesc        = "Synchronize subtitle using SubSync tool"
	dryRun             bool
	maxParallel        int
	streamLang         string
	streamType         string
	subtitleExtensions []string
	subtitleLang       string
	videoExtensions    []string
	videoLang          string
	yes                bool
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "subsync <directory>",
		Aliases: []string{"sub"},
		Short:   subsyncDesc,
		Long:    subsyncDesc + ".",
		Args:    cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			mediaSvc := ctxutil.Singleton[*mediasvc.Service](ctx)

			_, err := exec.LookPath(cmdutil.CommandSubsync)
			if err != nil {
				return fmt.Errorf("command not found: %s", cmdutil.CommandSubsync)
			}

			return mediaSvc.InitializeWD(args[0])
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			out := cmd.OutOrStdout()

			consoleSvc := ctxutil.Singleton[*consolesvc.Service](ctx)
			mediaSvc := ctxutil.Singleton[*mediasvc.Service](ctx)

			subtitleFiles := mediaSvc.List(config.WD, subtitleExtensions, nil)
			if len(subtitleFiles) == 0 {
				consoleSvc.Success("No subtitle file to process")
				return nil
			}

			videoFiles := mediaSvc.List(config.WD, videoExtensions, nil)
			if len(videoFiles) == 0 {
				consoleSvc.Success("No video file to process")
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

			err := process(ctx, out, videoFiles, subtitleFiles)
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
	cmd.PersistentFlags().StringArrayVar(&subtitleExtensions, "sub-ext", []string{"srt"}, "filter subtitles by extension")
	cmd.PersistentFlags().StringVar(&subtitleLang, "sub-lang", "eng", "subtitle ISO 639-3 language code")
	cmd.PersistentFlags().StringArrayVarP(&videoExtensions, "video-ext", "e", []string{"avi", "mkv", "mp4"}, "filter video files by extension")
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
	go pw.Render()

	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(cmdutil.MaxConcurrentGoroutines)
	if maxParallel > 0 {
		eg.SetLimit(maxParallel)
	}

	for i, v := range videoFiles {
		index, videoFile := i, v
		tracker := &progress.Tracker{
			DeferStart: true,
			Message:    fmt.Sprintf("%s%11s", videoFile, ""),
			Total:      100,
		}
		pw.AppendTracker(tracker)
		eg.Go(func() error {
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
