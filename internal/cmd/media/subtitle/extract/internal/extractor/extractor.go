package extractor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jedib0t/go-pretty/v6/progress"

	"github.com/jeremiergz/nas-cli/internal/config"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	_ svc.Runnable = (*process)(nil)
)

// SubtitleStream represents a subtitle track within an MKV file.
type SubtitleStream struct {
	// CodecName is the subtitle codec, e.g. "subrip", "ass".
	CodecName string
	// Encoding is the subtitle encoding, e.g. "UTF-8", "ISO-8859-1".
	Encoding string
	// Forced indicates whether this is a forced subtitle track.
	Forced bool
	// Lang is the ISO 639-2/3 language tag (e.g. "eng", "fre"). Falls back to "und".
	Lang string
	// SubtitleIndex is the 0-based position among subtitle streams (used as 0:s:N in ffmpeg).
	SubtitleIndex int
	// Title is the optional stream title tag, e.g. "English (SDH)".
	Title string
}

// OutputFilename returns the expected .srt filename for the given input file and language.
// Forced subtitle files include ".forced" before the extension.
func OutputFilename(inputFile, lang string, forced bool) string {
	basename := strings.TrimSuffix(inputFile, ".mkv")
	if forced {
		return fmt.Sprintf("%s.%s.forced.srt", basename, lang)
	}
	return fmt.Sprintf("%s.%s.srt", basename, lang)
}

type process struct {
	inputFile string
	streams   []SubtitleStream
	tracker   *progress.Tracker
	w         io.Writer
}

func New(inputFile string, streams []SubtitleStream) svc.Runnable {
	return &process{
		inputFile: inputFile,
		streams:   streams,
		w:         os.Stdout,
	}
}

func (p *process) Run(ctx context.Context) error {
	if p.tracker == nil {
		return fmt.Errorf("required tracker is not set")
	}

	p.tracker.Start()

	inputPath := filepath.Join(config.WD, p.inputFile)

	for i, stream := range p.streams {
		outFile := OutputFilename(p.inputFile, stream.Lang, stream.Forced)
		outPath := filepath.Join(config.WD, outFile)

		args := []string{
			"-y",
			"-i", inputPath,
			"-map", fmt.Sprintf("0:s:%d", stream.SubtitleIndex),
			outPath,
		}

		ffmpeg := exec.CommandContext(ctx, cmdutil.CommandFFmpeg, args...)

		bufOut := new(bytes.Buffer)
		bufErr := new(bytes.Buffer)
		ffmpeg.Stdout = bufOut
		ffmpeg.Stderr = bufErr

		if err := ffmpeg.Run(); err != nil {
			p.tracker.MarkAsErrored()
			return fmt.Errorf("failed to extract %s subtitle from %s: %w", stream.Lang, p.inputFile, err)
		}

		os.Chown(outPath, config.UID, config.GID)
		os.Chmod(outPath, config.FileMode)

		p.tracker.SetValue(int64((i + 1) * 100 / len(p.streams)))
	}

	p.tracker.MarkAsDone()
	return nil
}

func (p *process) SetTracker(tracker *progress.Tracker) svc.Runnable {
	p.tracker = tracker
	return p
}

func (p *process) SetOutput(out io.Writer) svc.Runnable {
	p.w = out
	return p
}
