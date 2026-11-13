package sync

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/cmd/media/subtitle/sync/internal/syncer"
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
	syncDesc          = "Synchronize subtitle using SubSync tool"
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
		Use:     "sync <directory>",
		Aliases: []string{"sc"},
		Short:   syncDesc,
		Long:    syncDesc + ".",
		Args:    cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmdutil.DebugMode {
				fmt.Fprintf(cmd.OutOrStdout(), "%s PreRunE\n", cmd.CommandPath())
			}

			_, err := exec.LookPath(cmdutil.CommandSubsync)
			if err != nil {
				return fmt.Errorf("command not found: %s", cmdutil.CommandSubsync)
			}

			_, err = exec.LookPath(cmdutil.CommandFFprobe)
			if err != nil {
				return fmt.Errorf("command not found: %s", cmdutil.CommandFFprobe)
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
				pterm.Success.Println("No subtitle file to process")
				return nil
			}

			videoFiles, err := media.Files(config.WD, videoExtensions, false)
			if err != nil {
				return err
			}
			if len(videoFiles) == 0 {
				pterm.Success.Println("No video file to process")
				return nil
			}

			sort.Slice(videoFiles, func(i, j int) bool {
				return strings.ToLower(videoFiles[i].Basename()) < strings.ToLower(videoFiles[j].Basename())
			})
			jobs := matchingSubtitles(videoFiles, subtitleExtension)
			if len(jobs) == 0 {
				pterm.Success.Println("No matching subtitle file to process")
				return nil
			}

			displayList(out, config.WD, jobs)
			if dryRun {
				return nil
			}

			var p prompt.Prompter
			if yes {
				p = prompt.NewAuto()
			} else {
				p = prompt.NewInteractive()
			}

			fmt.Fprintln(out)

			shouldProcess, err := p.Confirm("Process?", true)
			if err != nil {
				return nil
			}
			if !shouldProcess {
				return nil
			}

			fmt.Fprintln(out)

			err = process(cmd.Context(), out, jobs)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print result without processing it")
	cmd.Flags().IntVarP(&maxParallel, "max-parallel", "p", 0, "maximum number of parallel processes. 0 means no limit")
	cmd.Flags().StringVar(&streamLang, "stream", "", "prefer reference stream by ISO 639-3 language code")
	cmd.Flags().StringVar(&streamType, "stream-type", "audio", "reference stream type (audio|sub)")
	cmd.Flags().StringVar(&subtitleExtension, "sub-ext", util.AcceptedSubtitleExtension, "filter subtitles by extension")
	cmd.Flags().StringVar(&subtitleLang, "sub-lang", "", "subtitle ISO 639-3 language code (defaults to the subtitle filename or stream metadata)")
	cmd.Flags().StringArrayVarP(&videoExtensions, "video-ext", "e", util.AcceptedVideoExtensions, "filter video files by extension")
	cmd.Flags().StringVar(&videoLang, "video-lang", "", "reference stream ISO 639-3 language code (defaults to stream metadata)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "automatic yes to prompts")

	return cmd
}

func displayList(out io.Writer, wd string, jobs []syncJob) {
	lw := cmdutil.NewListWriter()
	lw.AppendItem(wd)
	for index, job := range jobs {
		lw.Indent()
		lw.AppendItem(index + 1)

		lw.Indent()
		lw.AppendItem(job.subtitle)
		lw.AppendItem(job.video)

		lw.UnIndentAll()
	}
	fmt.Fprintln(out, lw.Render())
}

type syncJob struct {
	video    string
	subtitle string
	forced   bool
}

func matchingSubtitles(videos []*media.File, extension string) []syncJob {
	var jobs []syncJob
	for _, video := range videos {
		subtitlesByLanguage := video.SubtitlesWithExtension(extension)
		languages := make([]string, 0, len(subtitlesByLanguage))
		for language := range subtitlesByLanguage {
			languages = append(languages, language)
		}
		sort.Strings(languages)

		for _, language := range languages {
			subtitles := subtitlesByLanguage[language]
			sort.Slice(subtitles, func(i, j int) bool {
				return strings.ToLower(subtitles[i].Name) < strings.ToLower(subtitles[j].Name)
			})
			for _, subtitle := range subtitles {
				jobs = append(jobs, syncJob{
					video:    video.Basename(),
					subtitle: subtitle.Name,
					forced:   subtitle.Kind == media.SubtitleKindForced,
				})
			}
		}
	}
	return jobs
}

func process(ctx context.Context, out io.Writer, jobs []syncJob) error {
	videoFiles := make([]string, len(jobs))
	streams := make([]referenceStream, len(jobs))
	streamsByVideo := make(map[string]referenceStream)
	for i, job := range jobs {
		videoFiles[i] = job.video
		stream, ok := streamsByVideo[job.video]
		if !ok {
			videoPath := filepath.Join(config.WD, job.video)
			var err error
			stream, err = probeReferenceStream(ctx, videoPath, streamType, streamLang)
			if err != nil {
				return fmt.Errorf("failed to select reference stream for %s: %w", job.video, err)
			}
			streamsByVideo[job.video] = stream
		}
		streams[i] = stream
	}

	pw := cmdutil.NewProgressWriter(out, len(jobs))

	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(cmdutil.MaxConcurrentGoroutines)
	if maxParallel > 0 {
		eg.SetLimit(maxParallel)
	}

	padder := str.NewPadder(videoFiles)

	syncers := make([]svc.Runnable, len(jobs))
	for index, job := range jobs {
		subLang := languageForSubtitle(job.subtitle, subtitleLang)
		paddingLength := padder.PaddingLength(job.video, 12)
		tracker := &progress.Tracker{
			DeferStart: true,
			Message:    fmt.Sprintf("%s%*s", job.video, paddingLength, " "),
			Total:      100,
		}
		pw.AppendTracker(tracker)

		s := syncer.
			New(
				job.video,
				normalizeSubsyncLanguage(firstNonEmpty(videoLang, streams[index].language)),
				job.subtitle,
				subLang,
				streams[index].index+1,
				outputSubtitleName(job.video, subLang, job.forced),
			).
			SetOutput(out).
			SetTracker(tracker)
		syncers[index] = s
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

func languageForSubtitle(subtitle, override string) string {
	if override != "" {
		return normalizeSubsyncLanguage(override)
	}

	name := strings.TrimSuffix(filepath.Base(subtitle), filepath.Ext(subtitle))
	if strings.EqualFold(filepath.Ext(name), ".forced") {
		name = strings.TrimSuffix(name, filepath.Ext(name))
	}

	languageStart := strings.LastIndex(name, ".")
	if languageStart < 0 {
		return ""
	}

	language := name[languageStart+1:]
	if len(language) != 3 {
		return ""
	}
	for _, char := range language {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') {
			return ""
		}
	}

	return normalizeSubsyncLanguage(language)
}

func outputSubtitleName(video, language string, forced bool) string {
	name := strings.TrimSuffix(video, filepath.Ext(video))
	if language != "" {
		name += "." + language
	}
	if forced {
		name += ".forced"
	}
	return name + ".srt"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeSubsyncLanguage(language string) string {
	language = strings.ToLower(strings.TrimSpace(language))
	if language == "fra" || language == "fr" {
		return "fre"
	}
	return language
}
