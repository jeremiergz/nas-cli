package subsync

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

type Process struct {
	tracker *progress.Tracker
	w       io.Writer
}

func New(tracker *progress.Tracker, out io.Writer) *Process {
	return &Process{
		tracker: tracker,
		w:       out,
	}
}

// Attempts to synchronize given subtitle with given video file.
func (p *Process) Run(video, videoLang, subtitle, subtitleLang, streamLang, streamType, outFile string) error {
	p.tracker.Start()

	videoPath := path.Join(config.WD, video)
	subtitlePath := path.Join(config.WD, subtitle)
	outFilePath := path.Join(config.WD, outFile)
	options := []string{
		"sync",
		"--ref",
		videoPath,
		"--ref-lang",
		videoLang,
		"--sub",
		subtitlePath,
		"--sub-lang",
		subtitleLang,
		"--out",
		outFilePath,
	}

	if configOptions := viper.GetStringSlice(config.KeySubsyncOptions); len(configOptions) > 0 {
		options = append(options, configOptions...)
	}

	if streamLang != "" {
		options = append(options, "--ref-stream-by-lang", streamLang)
	}

	if streamType != "" {
		options = append(options, "--ref-stream-by-type", streamType)
	}

	var buf bytes.Buffer
	subsync := exec.Command(cmdutil.CommandSubsync, options...)
	subsync.Stdout = &buf

	if err := subsync.Start(); err != nil {
		p.tracker.MarkAsErrored()
		return err
	}

	go func() {
		originalMessage := p.tracker.Message[:(len(p.tracker.Message) - 12)] // Remove margin from the message.
		for !p.tracker.IsDone() {
			progress, points, err := getProgress(buf.String())
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

var progressRegexp = regexp.MustCompile(`(?m)(?:progress\s+)(?P<Percentage>\d+)(?:%)(?:,\s+)(?P<Points>\d+)(?:\s+points)`)

func getProgress(str string) (percentage int, points int, err error) {
	allProgressMatches := progressRegexp.FindAllStringSubmatch(str, -1)
	if len(allProgressMatches) == 0 {
		return 0, 0, fmt.Errorf("could not find progress percentage and points")
	}

	progressMatches := allProgressMatches[len(allProgressMatches)-1]

	if len(progressMatches) != 3 {
		return 0, 0, fmt.Errorf("could not find progress percentage and points")
	}

	percentageIndex := progressRegexp.SubexpIndex("Percentage")
	if percentageIndex == -1 {
		return 0, 0, fmt.Errorf("could not determine progress percentage")
	}
	percentage, err = strconv.Atoi(progressMatches[percentageIndex])
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse progress percentage: %w", err)
	}

	pointsIndex := progressRegexp.SubexpIndex("Points")
	if pointsIndex == -1 {
		return 0, 0, fmt.Errorf("could not determine progress points")
	}
	points, err = strconv.Atoi(progressMatches[pointsIndex])
	if err != nil {
		return 0, 0, fmt.Errorf("could not parse progress points: %w", err)
	}

	return percentage, points, nil
}
