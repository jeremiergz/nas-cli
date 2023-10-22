package subsync

import (
	"fmt"
	"io"
	"os/exec"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/list"
	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/cmd/media/subsync/internal"
	"github.com/jeremiergz/nas-cli/config"
	consoleservice "github.com/jeremiergz/nas-cli/service/console"
	mediaservice "github.com/jeremiergz/nas-cli/service/media"
	"github.com/jeremiergz/nas-cli/util"
	"github.com/jeremiergz/nas-cli/util/cmdutil"
	"github.com/jeremiergz/nas-cli/util/ctxutil"
)

var (
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

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "subsync <directory>",
		Aliases: []string{"sub"},
		Short:   "Synchronize subtitle using SubSync tool",
		Args:    cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			mediaSvc := ctxutil.Singleton[*mediaservice.Service](ctx)

			_, err := exec.LookPath(cmdutil.CommandSubsync)
			if err != nil {
				return fmt.Errorf("command not found: %s", cmdutil.CommandSubsync)
			}

			return mediaSvc.InitializeWD(args[0])
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			out := cmd.OutOrStdout()

			consoleSvc := ctxutil.Singleton[*consoleservice.Service](ctx)
			mediaSvc := ctxutil.Singleton[*mediaservice.Service](ctx)

			if subtitleFiles := mediaSvc.List(config.WD, subtitleExtensions, nil); len(subtitleFiles) == 0 {
				consoleSvc.Success("No subtitle file to process")
			} else if videoFiles := mediaSvc.List(config.WD, videoExtensions, nil); len(videoFiles) == 0 {
				consoleSvc.Success("No video file to process")
			} else {
				sort.Sort(util.SortAlphabetic(videoFiles))
				sort.Sort(util.SortAlphabetic(subtitleFiles))

				printAll(out, videoFiles, subtitleFiles)

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

				err := process(videoFiles, subtitleFiles)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print result without processing it")
	cmd.Flags().IntVar(&maxParallel, "max-parallel", 3, "maximum number of parallel processes")
	cmd.Flags().StringVar(&streamLang, "stream", "eng", "stream ISO 639-3 language code")
	cmd.Flags().StringVar(&streamType, "stream-type", "", "stream type (audio|sub)")
	cmd.Flags().StringArrayVar(&subtitleExtensions, "sub-ext", []string{"srt"}, "filter subtitles by extension")
	cmd.Flags().StringVar(&subtitleLang, "sub-lang", "eng", "subtitle ISO 639-3 language code")
	cmd.Flags().StringArrayVarP(&videoExtensions, "video-ext", "e", []string{"avi", "mkv", "mp4"}, "filter video files by extension")
	cmd.Flags().StringVar(&videoLang, "video-lang", "eng", "video ISO 639-3 language code")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "automatic yes to prompts")

	return cmd
}

func printAll(out io.Writer, videos []string, subtitles []string) {
	tree := list.NewWriter()
	tree.SetStyle(list.StyleConnectedLight)
	tree.AppendItem(config.WD)
	for index, video := range videos {
		tree.Indent()
		tree.AppendItem(index + 1)

		tree.Indent()
		tree.AppendItem(subtitles[index])
		tree.AppendItem(video)

		tree.UnIndentAll()
	}
	fmt.Fprintln(out, tree.Render())
}

func process(videoFiles, subtitleFiles []string) error {
	pw := progress.NewWriter()
	pw.SetTrackerLength(25)
	pw.SetNumTrackersExpected(len(videoFiles))
	pw.SetSortBy(progress.SortByNone)
	pw.SetStyle(progress.StyleCircle)
	pw.SetTrackerPosition(progress.PositionRight)
	pw.SetUpdateFrequency(time.Millisecond * 100)
	pw.Style().Colors = progress.StyleColorsDefault
	pw.Style().Options.PercentFormat = "%3.0f%%"
	pw.Style().Options.TimeDonePrecision = time.Second
	pw.Style().Options.TimeInProgressPrecision = time.Second
	pw.Style().Options.TimeOverallPrecision = time.Second
	pw.Style().Visibility.Value = false
	pw.Style().Options.DoneString = fmt.Sprintf("  %s", util.StyleSuccess("✔"))
	pw.Style().Options.Separator = ""
	pw.Style().Chars.BoxLeft = ""
	pw.Style().Chars.BoxRight = ""
	go pw.Render()

	eg := errgroup.Group{}
	eg.SetLimit(maxParallel)

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
	err := eg.Wait()
	if err != nil {
		return err
	}

	return nil
}
