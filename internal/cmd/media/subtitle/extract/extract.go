package extract

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/pterm/pterm"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/cmd/media/subtitle/extract/internal/extractor"
	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/prompt"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/service/str"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	extractDesc = "Extract subtitle files"
	dryRun      bool
	maxParallel int
	yes         bool
)

type mkvEntry struct {
	file     string
	streams  []extractor.SubtitleStream
	duration int64 // Nanoseconds, from mkvmerge container properties.
}

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "extract <directory>",
		Aliases: []string{"ex"},
		Short:   extractDesc,
		Long:    extractDesc + ".",
		Args:    cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmdutil.DebugMode {
				fmt.Fprintf(cmd.OutOrStdout(), "%s PreRunE\n", cmd.CommandPath())
			}

			for _, c := range []string{cmdutil.CommandFFmpeg, cmdutil.CommandMKVMerge} {
				if _, err := exec.LookPath(c); err != nil {
					return fmt.Errorf("command not found: %s", c)
				}
			}

			selectedDir := "."
			if len(args) > 0 {
				selectedDir = args[0]
			}

			err := fsutil.InitializeWorkingDir(selectedDir)
			if err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			ctx := cmd.Context()

			mkvFiles := fsutil.List(config.WD, []string{util.ExtensionMKV}, nil, false)
			if len(mkvFiles) == 0 {
				fmt.Fprintln(out, "No MKV file to process")
				return nil
			}

			sort.Sort(util.SortAlphabetic(mkvFiles))

			var entries []mkvEntry
			for _, f := range mkvFiles {
				streams, durationNS, err := probeSubtitles(ctx, filepath.Join(config.WD, f))
				if err != nil {
					return err
				}
				if len(streams) > 0 {
					entries = append(entries, mkvEntry{file: f, streams: streams, duration: durationNS})
				}
			}

			if len(entries) == 0 {
				fmt.Fprintln(out, "No subtitle streams found")
				return nil
			}

			displayList(out, config.WD, entries)
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

			err = process(ctx, out, entries)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print result without processing it")
	cmd.Flags().IntVarP(&maxParallel, "max-parallel", "p", 0, "maximum number of parallel processes. 0 means no limit")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "automatic yes to prompts")

	return cmd
}

func displayList(out io.Writer, wd string, entries []mkvEntry) {
	lw := cmdutil.NewListWriter()
	lw.AppendItem(wd)
	for index, e := range entries {
		lw.Indent()
		lw.AppendItem(index + 1)

		lw.Indent()
		lw.AppendItem(e.file)
		maxLen := 0
		for _, s := range e.streams {
			if l := len(extractor.OutputFilename(e.file, s.Lang, s.Forced)); l > maxLen {
				maxLen = l
			}
		}
		for _, s := range e.streams {
			outFile := extractor.OutputFilename(e.file, s.Lang, s.Forced)
			info := fmt.Sprintf("#%d: %s - %s/%s", s.SubtitleIndex, s.Lang, s.CodecName, s.Encoding)
			if s.Forced {
				info += " - forced"
			}
			if s.Title != "" {
				info += " - " + s.Title
			}
			lw.AppendItem(fmt.Sprintf("%-*s  <-  %s", maxLen, outFile, pterm.Gray(info)))
		}

		lw.UnIndentAll()
	}
	fmt.Fprintln(out, lw.Render())
}

// Extracts subtitles from MKV files using ffmpeg.
func process(ctx context.Context, w io.Writer, entries []mkvEntry) error {
	pw := cmdutil.NewProgressWriter(w, len(entries))

	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(cmdutil.MaxConcurrentGoroutines)
	if maxParallel > 0 {
		eg.SetLimit(maxParallel)
	}

	padder := str.NewPadder(lo.Map(entries, func(e mkvEntry, _ int) string { return e.file }))

	extractors := make([]svc.Runnable, len(entries))
	for i, e := range entries {
		paddingLength := padder.PaddingLength(e.file, 1)
		tracker := &progress.Tracker{
			DeferStart: true,
			Message:    fmt.Sprintf("%s%*s", e.file, paddingLength, " "),
			Total:      100,
		}
		pw.AppendTracker(tracker)

		ex := extractor.
			New(e.file, e.streams, e.duration).
			SetOutput(w).
			SetTracker(tracker)
		extractors[i] = ex
	}
	for _, ex := range extractors {
		eg.Go(func() error {
			return ex.Run(ctx)
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

type mkvIdentification struct {
	Container *mkvContainer `json:"container,omitempty"`
	Tracks    []*mkvTrack   `json:"tracks"`
}

type mkvContainer struct {
	Properties *mkvContainerProperties `json:"properties,omitempty"`
}

type mkvContainerProperties struct {
	Duration int64 `json:"duration,omitempty"` // nanoseconds
}

type mkvTrack struct {
	Codec      string              `json:"codec"`
	Properties *mkvTrackProperties `json:"properties,omitempty"`
	Type       string              `json:"type"`
}

type mkvTrackProperties struct {
	Encoding    string `json:"encoding,omitempty"`
	ForcedTrack bool   `json:"forced_track"`
	Language    string `json:"language,omitempty"`
	TrackName   string `json:"track_name,omitempty"`
}

// probeSubtitles returns all subtitle streams (including forced) found in the given file,
// along with the container duration in nanoseconds (0 if unavailable).
func probeSubtitles(ctx context.Context, filePath string) ([]extractor.SubtitleStream, int64, error) {
	args := []string{
		"--identification-format", "json",
		"--identify",
		filePath,
	}

	out, err := exec.CommandContext(ctx, cmdutil.CommandMKVMerge, args...).Output()
	if err != nil {
		return nil, 0, fmt.Errorf("mkvmerge identify failed on %s: %w", filePath, err)
	}

	var result mkvIdentification
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, 0, fmt.Errorf("failed to parse mkvmerge output: %w", err)
	}

	var durationNS int64
	if result.Container != nil && result.Container.Properties != nil {
		durationNS = result.Container.Properties.Duration
	}

	var streams []extractor.SubtitleStream
	subtitleIndex := 0
	for _, track := range result.Tracks {
		if track.Type != "subtitles" {
			continue
		}
		isForced := track.Properties.ForcedTrack ||
			strings.Contains(strings.ToLower(track.Properties.TrackName), "forc")
		lang := track.Properties.Language
		if lang == "" {
			lang = "und"
		}
		streams = append(streams, extractor.SubtitleStream{
			CodecName:     track.Codec,
			Encoding:      track.Properties.Encoding,
			Forced:        isForced,
			Lang:          lang,
			SubtitleIndex: subtitleIndex,
			Title:         track.Properties.TrackName,
		})
		subtitleIndex++
	}

	return streams, durationNS, nil
}
