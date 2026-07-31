package merger

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
	"github.com/pterm/pterm"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/media"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	_ svc.Runnable = (*process)(nil)
)

type process struct {
	file         *media.File
	keepOriginal bool
	tracker      *progress.Tracker
	w            io.Writer
}

func New(file *media.File, keepOriginal bool) svc.Runnable {
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

	options, backups, removedPGS, err := computeMergeOptions(ctx, p.file.FilePath(), videoFileBackupPath, backups, subtitles)
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

	if removedPGS > 0 {
		p.tracker.UpdateMessage(strings.TrimRight(p.tracker.Message, " ") +
			fmt.Sprintf(
				" %s removed %d PGS subtitle(s)",
				pterm.FgYellow.Sprint("[!]"),
				removedPGS,
			))
	}

	p.tracker.MarkAsDone()
	return nil
}

// Holds forced and full subtitle track entries for a single language.
type subtitleGroup struct {
	forced []string
	full   []string
}

// Builds mkvmerge options and identifies tracks to compute --track-order.
func computeMergeOptions(
	ctx context.Context,
	videoFilePath string,
	videoFileBackupPath string,
	backups []backup,
	subtitles map[string][]media.Subtitle,
) ([]string, []backup, int, error) {
	// We'll assemble input-specific args separately so we can place global flags
	// like --track-order and --tracks before the input files.
	options := []string{"--gui-mode", "--output", videoFilePath}
	inputFiles := []string{}
	langByFile := map[string]string{}
	forcedFiles := map[string]bool{}
	inputArgs := []string{}

	addInput := func(lang, filename string, forced bool) {
		subtitleFilePath := path.Join(config.WD, filename)
		subtitleFileBackupPath := path.Join(config.WD, fmt.Sprintf("_%s.bak", filename))
		os.Rename(subtitleFilePath, subtitleFileBackupPath)
		backups = append(backups, backup{currentPath: subtitleFileBackupPath, originalPath: subtitleFilePath})
		langByFile[subtitleFileBackupPath] = lang
		forcedFiles[subtitleFileBackupPath] = forced
		inputFiles = append(inputFiles, subtitleFileBackupPath)
		if forced {
			inputArgs = append(inputArgs, "--language", fmt.Sprintf("0:%s", lang), "--forced-track", "0:1", subtitleFileBackupPath)
		} else {
			inputArgs = append(inputArgs, "--language", fmt.Sprintf("0:%s", lang), subtitleFileBackupPath)
		}
	}

	for lang, subs := range subtitles {
		for _, sub := range subs {
			addInput(lang, sub.Name, sub.Kind == media.SubtitleKindForced)
		}
	}

	// Video file is passed as last input.
	inputFiles = append(inputFiles, videoFileBackupPath)
	inputArgs = append(inputArgs, videoFileBackupPath)

	// Build per-kind sets of incoming languages so --override-language only replaces
	// the kinds that are actually being supplied (forced replaces forced, full replaces full).
	incomingForcedLangs := map[string]struct{}{}
	incomingFullLangs := map[string]struct{}{}
	for backupPath, lang := range langByFile {
		norm := normalizeLanguage(lang)
		if norm == "" {
			continue
		}
		if forcedFiles[backupPath] {
			incomingForcedLangs[norm] = struct{}{}
		} else {
			incomingFullLangs[norm] = struct{}{}
		}
	}

	// Build track-order by identifying each input and collecting track IDs.
	type identOut struct {
		Tracks []struct {
			Codec      string `json:"codec"`
			ID         int    `json:"id"`
			Type       string
			Properties struct {
				ForcedTrack  bool   `json:"forced_track"`
				Language     string `json:"language,omitempty"`
				LanguageIETF string `json:"language_ietf,omitempty"`
			} `json:"properties"`
		} `json:"tracks"`
	}

	nonSubtitle := []string{}
	frenchSubs := subtitleGroup{}
	englishSubs := subtitleGroup{}
	otherSubs := subtitleGroup{}
	// Collect subtitle track IDs to keep from the video input.
	videoSubtitleTrackIDsToKeep := []string{}
	videoIndex := len(inputFiles) - 1
	removedPGS := 0

	for idx, input := range inputFiles {
		idOpts := []string{"--identification-format", "json", "--identify", input}
		idCmd := exec.CommandContext(ctx, cmdutil.CommandMKVMerge, idOpts...)
		idOutBuf := new(bytes.Buffer)
		idErrBuf := new(bytes.Buffer)
		idCmd.Stdout = idOutBuf
		idCmd.Stderr = idErrBuf

		if err := idCmd.Run(); err != nil {
			return nil, backups, 0, util.ErrorFromStrings(fmt.Errorf("unable to identify input %s: %w", input, err), idOutBuf.String(), idErrBuf.String())
		}

		var id identOut
		if err := json.Unmarshal(idOutBuf.Bytes(), &id); err != nil {
			return nil, backups, 0, fmt.Errorf("unable to parse MKVMerge identification for %s: %w", input, err)
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

			// Drop PGS (bitmap) subtitle tracks from the video input.
			if idx == videoIndex && t.Codec == util.CodecPGS {
				removedPGS++
				continue
			}

			// Drop existing subtitle tracks of the same kind as the incoming ones for matching languages.
			if idx == videoIndex && norm != "" {
				if t.Properties.ForcedTrack {
					if _, ok := incomingForcedLangs[norm]; ok {
						continue
					}
				} else {
					if _, ok := incomingFullLangs[norm]; ok {
						continue
					}
				}
			}

			// Categorize subtitle track by language, separating forced from full.
			group := &otherSubs
			if isFrench(norm) {
				group = &frenchSubs
			} else if isEnglish(norm) {
				group = &englishSubs
			}
			if t.Properties.ForcedTrack || forcedFiles[input] {
				group.forced = append(group.forced, entry)
			} else {
				group.full = append(group.full, entry)
			}
			if idx == videoIndex {
				videoSubtitleTrackIDsToKeep = append(videoSubtitleTrackIDsToKeep, strconv.Itoa(t.ID))
			}
		}
	}

	// Final desired order: non-subtitle tracks, then subtitle groups by language
	// (French, English, other). Within each language group: forced tracks first, then full.
	finalOrder := append([]string{}, nonSubtitle...)
	finalOrder = append(finalOrder, frenchSubs.forced...)
	finalOrder = append(finalOrder, frenchSubs.full...)
	finalOrder = append(finalOrder, englishSubs.forced...)
	finalOrder = append(finalOrder, englishSubs.full...)
	// FIXME: other subtitles are currently all lumped together with no guaranteed order, which could lead to issues if
	// there are multiple "other" subtitle tracks. A more robust solution would be to preserve the original order of
	// "other" subtitle tracks while still grouping them after French and English. This would require a more complex data
	// structure to hold "other" subtitle tracks while still allowing us to place them after the French and English groups
	// in the final order.
	finalOrder = append(finalOrder, otherSubs.forced...)
	finalOrder = append(finalOrder, otherSubs.full...)

	if len(finalOrder) > 0 {
		options = append(options, "--track-order", strings.Join(finalOrder, ","))
	}

	// Limit subtitle tracks for the video input to the ones we kept.
	// This replaces only the incoming subtitle languages while preserving other subtitle tracks.
	{
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

	return options, backups, removedPGS, nil
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
