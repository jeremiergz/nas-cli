package clean

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	_ svc.Runnable = (*process)(nil)
)

type process struct {
	file         *model.File
	keepOriginal bool
	tracker      *progress.Tracker
	w            io.Writer
}

func New(file *model.File, keepOriginal bool) svc.Runnable {
	return &process{
		file:         file,
		keepOriginal: keepOriginal,
		w:            os.Stdout,
	}
}

func (p *process) Run() error {
	if p.tracker == nil {
		return fmt.Errorf("required tracker is not set")
	}

	p.tracker.Start()

	if p.file.Extension() != util.ExtensionMKV {
		err := p.convertToMKV()
		if err != nil {
			p.tracker.MarkAsErrored()
			return fmt.Errorf("failed to convert file to MKV: %w", err)
		}
	}

	err := p.cleanTracks()
	if err != nil {
		p.tracker.MarkAsErrored()
		return fmt.Errorf("failed to clean file: %w", err)
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

func (p *process) convertToMKV() error {
	originalFilePath := p.file.FilePath()
	originalFilePathWithoutExtension := strings.TrimSuffix(
		originalFilePath,
		fmt.Sprintf(".%s", p.file.Extension()),
	)
	newFilePath := fmt.Sprintf("%s.%s", originalFilePathWithoutExtension, util.ExtensionMKV)

	options := []string{
		"--gui-mode",
		"--output",
		newFilePath,
		originalFilePath,
	}

	var buf bytes.Buffer

	convert := exec.Command(cmdutil.CommandMKVMerge, options...)
	convert.Stdout = &buf
	convert.Stderr = p.w

	if err := convert.Start(); err != nil {
		return err
	}

	go func() {
		for !p.tracker.IsDone() {
			progress, err := cmdutil.GetMKVMergeProgress(buf.String())
			if err == nil {
				if progress == 100 {
					return
				}
				p.tracker.SetValue(int64(progress))
			}
			buf.Reset()
			time.Sleep(100 * time.Millisecond)
		}
	}()

	if err := convert.Wait(); err != nil {
		return err
	}

	os.Chown(newFilePath, config.UID, config.GID)
	os.Chmod(newFilePath, config.FileMode)

	if p.keepOriginal {
		backupFilePath := filepath.Join(
			filepath.Dir(originalFilePath),
			fmt.Sprintf("_%s.bak",
				filepath.Base(originalFilePath),
			),
		)
		os.Rename(originalFilePath, backupFilePath)
	} else {
		os.Remove(originalFilePath)
	}

	p.file.SetFilePath(newFilePath)

	return nil
}

func (p *process) cleanTracks() error {
	characteristics, err := getCharacteristics(p.file.FilePath())
	if err != nil {
		return err
	}

	options := []string{
		"--edit",
		"info",
		"--set",
		"title=",
		"--tags",
		"all:",
	}

	audioTrackNumber := 1
	subtitleTrackNumber := 1
	videoTrackNumber := 1

	for _, track := range characteristics.Tracks {
		switch track.Type {
		case "audio":
			options = append(options,
				"--edit",
				fmt.Sprintf("track:a%d", audioTrackNumber),
				"--set",
				fmt.Sprintf("language=%s", util.ToLanguageRegionalized(track.Properties.Language)),
				"--set",
				"name=",
			)
			audioTrackNumber++

		case "subtitles":
			isForced := false
			normalizedTrackName := strings.ToLower(track.Properties.TrackName)
			if track.Properties.ForcedTrack || strings.Contains(normalizedTrackName, "forc") {
				isForced = true
			}

			options = append(options,
				"--edit",
				fmt.Sprintf("track:s%d", subtitleTrackNumber),
				"--set",
				fmt.Sprintf("language=%s", util.ToLanguageRegionalized(track.Properties.Language)),
				"--set",
				fmt.Sprintf("name=%s", util.ToLanguageDisplayName(track.Properties.Language, isForced)),
			)
			if isForced {
				options = append(options,
					"--set",
					"flag-default=1",
					"--set",
					"flag-forced=1",
				)
			} else {
				options = append(options,
					"--set",
					"flag-default=0",
					"--set",
					"flag-forced=0",
				)
			}
			subtitleTrackNumber++

		case "video":
			options = append(options,
				"--edit",
				fmt.Sprintf("track:v%d", videoTrackNumber),
				"--set",
				"language=und",
				"--set",
				"name=",
			)
			videoTrackNumber++
		}
	}

	options = append(options, p.file.FilePath())

	propedit := exec.Command(cmdutil.CommandMKVPropEdit, options...)

	err = propedit.Run()
	if err != nil {
		return err
	}

	return nil
}

// Retrieves the characteristics of given MKV file.
func getCharacteristics(filePath string) (*mkvmergeIdentificationOutput, error) {
	options := []string{
		"--identification-format",
		"json",
		"--identify",
		filePath,
	}

	buf := new(bytes.Buffer)

	merge := exec.Command(cmdutil.CommandMKVMerge, options...)
	merge.Stdout = buf
	merge.Stderr = os.Stderr
	err := merge.Run()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve characteristics: %w", err)
	}

	var characteristics *mkvmergeIdentificationOutput
	err = json.Unmarshal(buf.Bytes(), &characteristics)
	if err != nil {
		return nil, fmt.Errorf("unable to parse characteristics: %w", err)
	}

	return characteristics, nil
}
