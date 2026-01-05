package mkvmerge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
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

type backup struct {
	currentPath  string
	originalPath string
}

func (p *process) Run(ctx context.Context) error {
	if p.tracker == nil {
		return fmt.Errorf("required tracker is not set")
	}

	p.tracker.Start()

	subtitles := p.file.Subtitles()
	if len(subtitles) == 0 {
		p.tracker.MarkAsDone()
		return nil
	}

	videoFileBackupPath := filepath.Join(config.WD, fmt.Sprintf("_%s.bak", p.file.Basename()))

	err := os.Rename(p.file.FilePath(), videoFileBackupPath)
	if err != nil {
		p.tracker.MarkAsErrored()
		return fmt.Errorf("failed to rename video file: %w", err)
	}
	backups := []backup{
		{currentPath: videoFileBackupPath, originalPath: p.file.FilePath()},
	}

	options, backups, err := computeMergeOptions(ctx, p.file.FilePath(), videoFileBackupPath, backups, subtitles)
	if err != nil {
		// Restore backups.
		wg := sync.WaitGroup{}
		for _, b := range backups {
			wg.Add(1)
			go func(b backup) {
				defer wg.Done()
				os.Rename(b.currentPath, b.originalPath)
			}(b)
		}
		wg.Wait()
		p.tracker.MarkAsErrored()
		return err
	}
	merge := exec.CommandContext(ctx, cmdutil.CommandMKVMerge, options...)

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	merge.Stdout = bufOut
	merge.Stderr = bufErr

	if err = merge.Start(); err != nil {
		p.tracker.MarkAsErrored()
		return err
	}

	go func() {
		for !p.tracker.IsDone() {
			progress, err := cmdutil.GetMKVMergeProgress(bufOut.String())
			if err == nil {
				p.tracker.SetValue(int64(progress))
			}
			bufOut.Reset()
			time.Sleep(100 * time.Millisecond)
		}
	}()

	if err = merge.Wait(); err != nil {
		wg := sync.WaitGroup{}
		for _, backupFile := range backups {
			wg.Add(1)
			go func(b backup) {
				defer wg.Done()
				os.Rename(b.currentPath, b.originalPath)
			}(backupFile)
		}
		wg.Wait()
		p.tracker.MarkAsErrored()
		return util.ErrorFromStrings(
			fmt.Errorf("failed to run MKVMerge: %w", err),
			bufOut.String(),
			bufErr.String(),
		)
	}

	os.Chown(p.file.FilePath(), config.UID, config.GID)
	os.Chmod(p.file.FilePath(), config.FileMode)

	if !p.keepOriginal {
		wg := sync.WaitGroup{}
		for _, backupFile := range backups {
			wg.Add(1)
			go func(b backup) {
				defer wg.Done()
				os.Remove(b.currentPath)
			}(backupFile)
		}
		wg.Wait()
	}

	p.tracker.MarkAsDone()
	return nil
}

// Builds mkvmerge options and identifies tracks to compute --track-order.
func computeMergeOptions(
	ctx context.Context,
	videoFilePath string,
	videoFileBackupPath string,
	backups []backup,
	subtitles map[string]string,
) ([]string, []backup, error) {
	options := []string{"--gui-mode", "--output", videoFilePath}

	// Keep track of input files order and their language (for subtitle inputs).
	inputFiles := []string{}
	langByFile := map[string]string{}

	for lang, subtitle := range subtitles {
		subtitleFilePath := path.Join(config.WD, subtitle)
		subtitleFileBackupPath := path.Join(config.WD, fmt.Sprintf("%s%s%s", "_", subtitle, ".bak"))
		os.Rename(subtitleFilePath, subtitleFileBackupPath)
		backups = append(backups, backup{currentPath: subtitleFileBackupPath, originalPath: subtitleFilePath})
		// Record the language for this input file so we can use it if MKVMerge identification doesn't include it.
		langByFile[subtitleFileBackupPath] = lang
		inputFiles = append(inputFiles, subtitleFileBackupPath)
		options = append(options, "--language", fmt.Sprintf("0:%s", lang), subtitleFileBackupPath)
	}

	// Video file is passed as last input.
	inputFiles = append(inputFiles, videoFileBackupPath)
	options = append(options, videoFileBackupPath)

	// Build track-order by identifying each input and collecting track IDs.
	type identOut struct {
		Tracks []struct {
			ID         int `json:"id"`
			Type       string
			Properties struct {
				Language     string `json:"language,omitempty"`
				LanguageIETF string `json:"language_ietf,omitempty"`
			} `json:"properties,omitempty"`
		} `json:"tracks"`
	}

	nonSubtitle := []string{}
	french := []string{}
	english := []string{}
	others := []string{}

	for idx, input := range inputFiles {
		idOpts := []string{"--identification-format", "json", "--identify", input}
		idCmd := exec.CommandContext(ctx, cmdutil.CommandMKVMerge, idOpts...)
		idOutBuf := new(bytes.Buffer)
		idErrBuf := new(bytes.Buffer)
		idCmd.Stdout = idOutBuf
		idCmd.Stderr = idErrBuf

		if err := idCmd.Run(); err != nil {
			return nil, backups, util.ErrorFromStrings(fmt.Errorf("unable to identify input %s: %w", input, err), idOutBuf.String(), idErrBuf.String())
		}

		var id identOut
		if err := json.Unmarshal(idOutBuf.Bytes(), &id); err != nil {
			return nil, backups, fmt.Errorf("unable to parse MKVMerge identification for %s: %w", input, err)
		}

		for _, t := range id.Tracks {
			entry := fmt.Sprintf("%d:%d", idx, t.ID)
			if t.Type != "subtitles" {
				nonSubtitle = append(nonSubtitle, entry)
				continue
			}

			lang := strings.ToLower(strings.TrimSpace(t.Properties.Language))
			if lang == "" {
				lang = strings.ToLower(strings.TrimSpace(t.Properties.LanguageIETF))
			}
			if lang == "" {
				// Fallback to the language we recorded when renaming files.
				if l, ok := langByFile[input]; ok {
					lang = strings.ToLower(strings.TrimSpace(l))
				}
			}

			// Normalize: prefer 3-letter codes when possible.
			norm := ""
			if len(lang) >= 3 {
				norm = lang[:3]
			} else if len(lang) == 2 {
				switch lang {
				case "fr":
					norm = "fre"
				case "en":
					norm = "eng"
				default:
					norm = lang
				}
			} else {
				norm = lang
			}

			switch norm {
			case "fre", "fra", "fr-":
				french = append(french, entry)
			case "eng", "en-":
				english = append(english, entry)
			default:
				others = append(others, entry)
			}
		}
	}

	// Final desired order: non-subtitle tracks, French subtitles, other subtitles, English subtitles.
	finalOrder := append([]string{}, nonSubtitle...)
	finalOrder = append(finalOrder, french...)
	finalOrder = append(finalOrder, others...)
	finalOrder = append(finalOrder, english...)

	if len(finalOrder) > 0 {
		options = append(options, "--track-order", strings.Join(finalOrder, ","))
	}

	return options, backups, nil
}

func (p *process) SetTracker(tracker *progress.Tracker) svc.Runnable {
	p.tracker = tracker
	return p
}

func (p *process) SetOutput(out io.Writer) svc.Runnable {
	p.w = out
	return p
}
