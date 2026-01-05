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
	"strconv"
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
	file             *model.File
	keepOriginal     bool
	overrideLanguage bool
	tracker          *progress.Tracker
	w                io.Writer
}

func New(file *model.File, keepOriginal bool, overrideLanguage bool) svc.Runnable {
	return &process{
		file:             file,
		keepOriginal:     keepOriginal,
		overrideLanguage: overrideLanguage,
		w:                os.Stdout,
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

	options, backups, err := computeMergeOptions(ctx, p.file.FilePath(), videoFileBackupPath, backups, subtitles, p.overrideLanguage)
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
	overrideLanguage bool,
) ([]string, []backup, error) {
	// We'll assemble input-specific args separately so we can place global flags
	// like --track-order and --tracks before the input files.
	options := []string{"--gui-mode", "--output", videoFilePath}
	inputFiles := []string{}
	langByFile := map[string]string{}
	inputArgs := []string{}

	for lang, subtitle := range subtitles {
		subtitleFilePath := path.Join(config.WD, subtitle)
		subtitleFileBackupPath := path.Join(config.WD, fmt.Sprintf("%s%s%s", "_", subtitle, ".bak"))
		os.Rename(subtitleFilePath, subtitleFileBackupPath)
		backups = append(backups, backup{currentPath: subtitleFileBackupPath, originalPath: subtitleFilePath})
		// Record the language for this input file so we can use it if MKVMerge identification doesn't include it.
		langByFile[subtitleFileBackupPath] = lang
		inputFiles = append(inputFiles, subtitleFileBackupPath)
		inputArgs = append(inputArgs, "--language", fmt.Sprintf("0:%s", lang), subtitleFileBackupPath)
	}

	// Video file is passed as last input.
	inputFiles = append(inputFiles, videoFileBackupPath)
	inputArgs = append(inputArgs, videoFileBackupPath)

	// Build a set of normalized incoming subtitle language codes (first 3 chars, lowercase).
	incomingLangs := map[string]struct{}{}
	for _, l := range langByFile {
		norm := normalizeLanguage(l)
		if norm != "" {
			incomingLangs[norm] = struct{}{}
		}
	}

	// Build track-order by identifying each input and collecting track IDs.
	type identOut struct {
		Tracks []struct {
			ID         int `json:"id"`
			Type       string
			Properties struct {
				Language     string `json:"language,omitempty"`
				LanguageIETF string `json:"language_ietf,omitempty"`
			} `json:"properties"`
		} `json:"tracks"`
	}

	nonSubtitle := []string{}
	frenchSubs := []string{}
	englishSubs := []string{}
	otherSubs := []string{}
	// Collect subtitle track IDs to keep from the video input.
	videoSubtitleTrackIDsToKeep := []string{}
	videoIndex := len(inputFiles) - 1

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

			// Determine the track's language.
			lang := t.Properties.Language
			if lang == "" {
				lang = t.Properties.LanguageIETF
			}
			if lang == "" {
				// Fallback to the language we recorded when renaming files.
				if l, ok := langByFile[input]; ok {
					lang = l
				}
			}
			norm := normalizeLanguage(lang)

			// If overrideLanguage is set and this is the video input, skip subtitle tracks
			// whose normalized language matches an incoming subtitle language.
			if overrideLanguage && idx == videoIndex {
				if norm != "" {
					if _, ok := incomingLangs[norm]; ok {
						continue
					}
				}
			}

			// Categorize subtitle track by language.
			if isFrench(norm) {
				frenchSubs = append(frenchSubs, entry)
			} else if isEnglish(norm) {
				englishSubs = append(englishSubs, entry)
			} else {
				otherSubs = append(otherSubs, entry)
			}
			if idx == videoIndex {
				videoSubtitleTrackIDsToKeep = append(videoSubtitleTrackIDsToKeep, strconv.Itoa(t.ID))
			}
		}
	}

	// Final desired order: non-subtitle tracks, French subtitles, other subtitles, English subtitles.
	finalOrder := append([]string{}, nonSubtitle...)
	finalOrder = append(finalOrder, frenchSubs...)
	finalOrder = append(finalOrder, englishSubs...)
	finalOrder = append(finalOrder, otherSubs...)

	if len(finalOrder) > 0 {
		options = append(options, "--track-order", strings.Join(finalOrder, ","))
	}

	// If --override-language is set, limit subtitle tracks for the video input to the ones we kept.
	// This replaces only the incoming subtitle languages while preserving other subtitle tracks.
	if overrideLanguage {
		// We must apply input-specific options *before* the file they apply to, so we
		// prepend them to inputArgs for the video input.
		videoArg := videoFileBackupPath
		inputArgs = inputArgs[:len(inputArgs)-1]
		if len(videoSubtitleTrackIDsToKeep) == 0 {
			inputArgs = append(inputArgs, "--no-subtitles")
		} else {
			inputArgs = append(inputArgs, "--subtitle-tracks", strings.Join(videoSubtitleTrackIDsToKeep, ","))
		}
		inputArgs = append(inputArgs, videoArg)
	}

	// Finally, append input-specific args (languages and file paths).
	options = append(options, inputArgs...)

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

// Returns a lowercase language code. If the input is 3+ chars, returns the first 3;
// otherwise returns the input as-is (lowercased). This allows agnostic language matching.
func normalizeLanguage(lang string) string {
	l := strings.ToLower(strings.TrimSpace(lang))
	if len(l) >= 3 {
		return l[:3]
	}
	return l
}

// Returns true if the normalized language code represents French.
func isFrench(norm string) bool {
	return strings.HasPrefix(norm, "fr")
}

// Returns true if the normalized language code represents English.
func isEnglish(norm string) bool {
	return strings.HasPrefix(norm, "en")
}
