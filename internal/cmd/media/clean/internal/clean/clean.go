package clean

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

func (p *process) Run(ctx context.Context) error {
	if p.tracker == nil {
		return fmt.Errorf("required tracker is not set")
	}

	p.tracker.Start()

	if p.file.Extension() != util.ExtensionMKV {
		err := p.convertToMKV(ctx)
		if err != nil {
			p.tracker.MarkAsErrored()
			return fmt.Errorf("failed to convert file to MKV: %w", err)
		}
	}

	err := p.cleanTracks(ctx)
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

func (p *process) convertToMKV(ctx context.Context) error {
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

	convert := exec.CommandContext(ctx, cmdutil.CommandMKVMerge, options...)

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	convert.Stdout = bufOut
	convert.Stderr = bufErr

	if err := convert.Start(); err != nil {
		return err
	}

	go func() {
		for !p.tracker.IsDone() {
			progress, err := cmdutil.GetMKVMergeProgress(bufOut.String())
			if err == nil {
				if progress == 100 {
					return
				}
				p.tracker.SetValue(int64(progress))
			}
			bufOut.Reset()
			time.Sleep(100 * time.Millisecond)
		}
	}()

	if err := convert.Wait(); err != nil {
		return util.ErrorFromStrings(
			fmt.Errorf("failed to convert file to MKV: %w", err),
			bufOut.String(),
			bufErr.String(),
		)
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

func (p *process) cleanTracks(ctx context.Context) error {
	characteristics, err := p.getCharacteristics(ctx)
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
		lang := util.ToLanguageRegionalized(
			cmp.Or(track.Properties.LanguageIETF, track.Properties.Language),
		)

		switch track.Type {
		case "audio":
			options = append(options,
				"--edit",
				fmt.Sprintf("track:a%d", audioTrackNumber),
				"--set",
				fmt.Sprintf("language=%s", lang),
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
				fmt.Sprintf("language=%s", lang),
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

	for _, attachment := range characteristics.Attachments {
		options = append(options,
			"--delete-attachment",
			strconv.Itoa(attachment.ID),
		)
	}

	options = append(options, p.file.FilePath())

	propedit := exec.CommandContext(ctx, cmdutil.CommandMKVPropEdit, options...)

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	propedit.Stdout = bufOut
	propedit.Stderr = bufErr

	err = propedit.Run()
	if err != nil {
		return util.ErrorFromStrings(
			fmt.Errorf("failed to run MKVPropEdit: %w", err),
			bufOut.String(),
			bufErr.String(),
		)
	}

	return nil
}

// Retrieves the characteristics of given MKV file.
func (p *process) getCharacteristics(ctx context.Context) (*mkvmergeIdentificationOutput, error) {
	options := []string{
		"--identification-format",
		"json",
		"--identify",
		p.file.FilePath(),
	}

	merge := exec.CommandContext(ctx, cmdutil.CommandMKVMerge, options...)

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	merge.Stdout = bufOut
	merge.Stderr = bufErr

	err := merge.Run()
	if err != nil {
		return nil, util.ErrorFromStrings(
			fmt.Errorf("unable to retrieve characteristics: %w", err),
			bufOut.String(),
			bufErr.String(),
		)
	}

	var characteristics *mkvmergeIdentificationOutput
	err = json.Unmarshal(bufOut.Bytes(), &characteristics)
	if err != nil {
		return nil, fmt.Errorf("unable to parse characteristics: %w", err)
	}

	return characteristics, nil
}
