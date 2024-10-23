package subsync

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
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
func (p *process) Run() error {
	if p.tracker == nil {
		return fmt.Errorf("required tracker is not set")
	}

	p.tracker.Start()

	videoPath := filepath.Join(config.WD, p.video)
	subtitlePath := filepath.Join(config.WD, p.subtitle)
	outFilePath := filepath.Join(config.WD, p.outFile)
	options := []string{
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

	var buf bytes.Buffer
	subsync := exec.Command(cmdutil.CommandSubsync, options...)
	subsync.Stdout = &buf
	if cmdutil.DebugMode {
		subsync.Stderr = p.w
	}

	if err := subsync.Start(); err != nil {
		p.tracker.MarkAsErrored()
		return err
	}

	go func() {
		originalMessage := p.tracker.Message[:(len(p.tracker.Message) - 12)] // Remove margin from the message.
		for !p.tracker.IsDone() {
			progress, points, err := cmdutil.GetSubsyncProgress(buf.String())
			if err == nil {
				p.tracker.SetValue(int64(progress))
				p.tracker.UpdateMessage(fmt.Sprintf("%s %s points ", originalMessage, formatPoints(points)))
			}
			buf.Reset()
			time.Sleep(100 * time.Millisecond)
		}
	}()

	if err := subsync.Wait(); err != nil {
		p.tracker.MarkAsErrored()
		return err
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
	var style func(interface{}) string
	if points < 30 {
		style = util.StyleError
	} else if points < 60 {
		style = util.StyleWarning
	} else {
		style = util.StyleSuccess
	}

	return style(fmt.Sprintf("%3d", points))
}
