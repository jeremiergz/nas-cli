package subsync

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/pterm/pterm"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/internal/config"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	_ svc.Runnable = (*process)(nil)
)

type process struct {
	outFile      string
	streamLang   string
	streamType   string
	subtitle     string
	subtitleLang string
	tracker      *progress.Tracker
	video        string
	videoLang    string
	w            io.Writer
}

func New(video, videoLang, subtitle, subtitleLang, streamLang, streamType, outFile string) svc.Runnable {
	return &process{
		outFile:      outFile,
		streamLang:   streamLang,
		streamType:   streamType,
		subtitle:     subtitle,
		subtitleLang: subtitleLang,
		video:        video,
		videoLang:    videoLang,
		w:            os.Stdout,
	}
}

// Attempts to synchronize given subtitle with given video file.
func (p *process) Run(ctx context.Context) error {
	if p.tracker == nil {
		return fmt.Errorf("required tracker is not set")
	}

	p.tracker.Start()

	videoPath := filepath.Join(config.WD, p.video)
	subtitlePath := filepath.Join(config.WD, p.subtitle)
	outFilePath := filepath.Join(config.WD, p.outFile)
	options := []string{
		"--cli",
		"sync",
		"--ref",
		videoPath,
		"--ref-lang",
		p.videoLang,
		"--sub",
		subtitlePath,
		"--sub-lang",
		p.subtitleLang,
		"--out",
		outFilePath,
	}

	if configOptions := viper.GetStringSlice(config.KeySubsyncOptions); len(configOptions) > 0 {
		options = append(options, configOptions...)
	}

	if p.streamLang != "" {
		options = append(options, "--ref-stream-by-lang", p.streamLang)
	}

	if p.streamType != "" {
		options = append(options, "--ref-stream-by-type", p.streamType)
	}

	subsync := exec.CommandContext(ctx, cmdutil.CommandSubsync, options...)

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	subsync.Stdout = bufOut
	subsync.Stderr = bufErr

	if err := subsync.Start(); err != nil {
		p.tracker.MarkAsErrored()
		return err
	}

	go func() {
		originalMessage := p.tracker.Message[:(len(p.tracker.Message) - 12)] // Remove margin from the message.
		for !p.tracker.IsDone() {
			progress, points, err := getSubsyncProgress(bufOut.String())
			if err == nil {
				p.tracker.SetValue(int64(progress))
				p.tracker.UpdateMessage(fmt.Sprintf("%s %s points ", originalMessage, formatPoints(points)))
			}
			bufOut.Reset()
			time.Sleep(100 * time.Millisecond)
		}
	}()

	if err := subsync.Wait(); err != nil {
		p.tracker.MarkAsErrored()
		return util.ErrorFromStrings(
			fmt.Errorf("failed to run Subsync: %w", err),
			bufOut.String(),
			bufErr.String(),
		)
	}

	os.Chown(outFilePath, config.UID, config.GID)
	os.Chmod(outFilePath, config.FileMode)

	p.tracker.MarkAsDone()
	return nil
}

func (p *process) SetOutput(w io.Writer) svc.Runnable {
	p.w = w
	return p
}

func (p *process) SetTracker(tracker *progress.Tracker) svc.Runnable {
	p.tracker = tracker
	return p
}

func formatPoints(points int) string {
	var style func(...any) string
	if points < 30 {
		style = pterm.Red
	} else if points < 60 {
		style = pterm.Yellow
	} else {
		style = pterm.Green
	}

	return style(fmt.Sprintf("%3d", points))
}

var subsyncProgressRegexp = regexp.MustCompile(`(?m)(?:progress\s+)(?P<Percentage>\d+)(?:%)(?:,\s+)(?P<Points>\d+)(?:\s+points)`)

func getSubsyncProgress(str string) (percentage int, points int, err error) {
	allProgressMatches := subsyncProgressRegexp.FindAllStringSubmatch(str, -1)
	if len(allProgressMatches) == 0 {
		return 0, 0, fmt.Errorf("could not find progress percentage and points")
	}

	progressMatches := allProgressMatches[len(allProgressMatches)-1]

	if len(progressMatches) != 3 {
		return 0, 0, fmt.Errorf("could not find progress percentage and points")
	}

	percentageIndex := subsyncProgressRegexp.SubexpIndex("Percentage")
	if percentageIndex == -1 {
		return 0, 0, fmt.Errorf("could not determine progress percentage")
	}
	percentage, err = strconv.Atoi(progressMatches[percentageIndex])
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse progress percentage: %w", err)
	}

	pointsIndex := subsyncProgressRegexp.SubexpIndex("Points")
	if pointsIndex == -1 {
		return 0, 0, fmt.Errorf("could not determine progress points")
	}
	points, err = strconv.Atoi(progressMatches[pointsIndex])
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse progress points: %w", err)
	}

	return percentage, points, nil
}
