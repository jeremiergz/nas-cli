package subcleaner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/asticode/go-astisub"
	"github.com/jedib0t/go-pretty/v6/progress"

	svc "github.com/jeremiergz/nas-cli/internal/service"
)

var (
	_ svc.Runnable = (*process)(nil)
)

type process struct {
	filePath     string
	keepOriginal bool
	tracker      *progress.Tracker
	w            io.Writer
}

func New(filePath string, keepOriginal bool) svc.Runnable {
	return &process{
		filePath:     filePath,
		keepOriginal: keepOriginal,
		w:            os.Stdout,
	}
}

type backup struct {
	currentPath  string
	originalPath string
}

var cleanupPipeline = []func([]*astisub.Item) []*astisub.Item{
	mergeDuplicateTimestamps,
	removeSDH,
	removeHTMLTags,
	removeEmptyItems,
}

func (p *process) Run(ctx context.Context) error {
	if p.tracker == nil {
		return fmt.Errorf("required tracker is not set")
	}

	p.tracker.Start()

	file, err := os.Open(p.filePath)
	if err != nil {
		p.tracker.MarkAsErrored()
		return err
	}
	defer file.Close()

	currentSubs, err := astisub.ReadFromSRT(file)
	if err != nil {
		p.tracker.MarkAsErrored()
		return err
	}

	cleanedItems := currentSubs.Items
	for _, cleanup := range cleanupPipeline {
		cleanedItems = cleanup(cleanedItems)
	}

	subs := astisub.NewSubtitles()
	subs.Items = cleanedItems

	backups, err := writeToSRTFile(subs, p.filePath)
	if err != nil {
		p.tracker.MarkAsErrored()
		return err
	}

	if !p.keepOriginal {
		wg := sync.WaitGroup{}
		for _, backupFile := range backups {
			wg.Go(func() { os.Remove(backupFile.currentPath) })
		}
		wg.Wait()
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

func mergeDuplicateTimestamps(items []*astisub.Item) []*astisub.Item {
	result := []*astisub.Item{}

	for _, item := range items {
		if len(result) > 0 {
			lastItem := result[len(result)-1]
			if lastItem.StartAt == item.StartAt && lastItem.EndAt == item.EndAt {
				lastItem.Lines = append(lastItem.Lines, item.Lines...)
				continue
			}
		}
		result = append(result, item)
	}

	return result
}

var (
	// Matches SDH content: text in brackets [like this] or parentheses (like this).
	sdhPattern = regexp.MustCompile(`\[.*?\]|\(.*?\)`)

	// Matches ASS/SSA override tags such as {\an8}.
	stylingTagPattern = regexp.MustCompile(`\{\\[^}]*\}`)

	// Matches music symbols (♪, ♫) and surrounding whitespace.
	musicPattern = regexp.MustCompile(`[♪♫]+`)

	// Matches speaker labels like "SPEAKER:" at the start of text.
	// colonPrefixPattern = regexp.MustCompile(`(?i)^[A-Z][A-Z0-9 ]*:\s*`)

	// Matches two or more consecutive spaces.
	multiSpacePattern = regexp.MustCompile(`  +`)

	// Matches HTML tags except <i> and </i>.
	htmlTagPattern = regexp.MustCompile(`</?(?:b|u|s|font|span|div|p)\b[^>]*>`)
)

// Strips SDH (Subtitles for the Deaf and Hard of Hearing) content
// from subtitle items.
//
// It removes bracketed/parenthesized annotations, music symbols, ASS/SSA styling tags, and uppercase speaker labels.
// Lines and items that become empty after stripping are discarded.
func removeSDH(items []*astisub.Item) []*astisub.Item {
	result := []*astisub.Item{}

	for _, item := range items {
		var cleanedLines []astisub.Line

		for _, line := range item.Lines {
			var cleanedLineItems []astisub.LineItem

			for _, lineItem := range line.Items {
				text := lineItem.Text

				// Remove bracketed and parenthesized SDH annotations.
				text = sdhPattern.ReplaceAllString(text, "")

				// Remove music symbols.
				text = musicPattern.ReplaceAllString(text, "")

				// Remove speaker labels (e.g. "NARRATOR:", "MAN 1:").
				// text = colonPrefixPattern.ReplaceAllString(text, "")

				// Collapse multiple consecutive spaces left after removals.
				text = multiSpacePattern.ReplaceAllString(text, " ")

				if strings.TrimSpace(text) == "" {
					continue
				}

				cleanedLineItem := lineItem
				cleanedLineItem.Text = text
				cleanedLineItems = append(cleanedLineItems, cleanedLineItem)
			}

			if len(cleanedLineItems) == 0 {
				continue
			}

			// Trim leading whitespace on the first item and trailing whitespace on the last item
			// while preserving inter-item spacing so that adjacent styled spans (e.g. italic)
			// keep their surrounding spaces.
			cleanedLineItems[0].Text = strings.TrimLeft(cleanedLineItems[0].Text, " \t")
			last := len(cleanedLineItems) - 1
			cleanedLineItems[last].Text = strings.TrimRight(cleanedLineItems[last].Text, " \t")

			cleanedLines = append(cleanedLines, astisub.Line{
				Items:     cleanedLineItems,
				VoiceName: line.VoiceName,
			})
		}

		if len(cleanedLines) == 0 {
			continue
		}

		cleanedItem := *item
		cleanedItem.Lines = cleanedLines
		result = append(result, &cleanedItem)
	}

	return result
}

// Strips HTML styling tags (e.g. <font>, <b>, <u>) from subtitle
// items while preserving <i> and </i> tags.
//
// Lines that become empty after stripping are discarded.
func removeHTMLTags(items []*astisub.Item) []*astisub.Item {
	result := []*astisub.Item{}

	for _, item := range items {
		var cleanedLines []astisub.Line

		for _, line := range item.Lines {
			var cleanedLineItems []astisub.LineItem

			for _, lineItem := range line.Items {
				text := htmlTagPattern.ReplaceAllString(lineItem.Text, "")

				if strings.TrimSpace(text) == "" {
					continue
				}

				cleanedLineItem := lineItem
				cleanedLineItem.Text = text
				if cleanedLineItem.InlineStyle != nil {
					// Preserve italic styling while stripping other HTML styles.
					preserved := &astisub.StyleAttributes{
						SRTItalics: cleanedLineItem.InlineStyle.SRTItalics,
					}
					if !preserved.SRTItalics {
						preserved = nil
					}
					cleanedLineItem.InlineStyle = preserved
				}
				cleanedLineItems = append(cleanedLineItems, cleanedLineItem)
			}

			if len(cleanedLineItems) == 0 {
				continue
			}

			// Trim leading/trailing whitespace only at line boundaries to preserve inter-item
			// spacing for adjacent styled spans.
			cleanedLineItems[0].Text = strings.TrimLeft(cleanedLineItems[0].Text, " \t")
			last := len(cleanedLineItems) - 1
			cleanedLineItems[last].Text = strings.TrimRight(cleanedLineItems[last].Text, " \t")

			cleanedLines = append(cleanedLines, astisub.Line{
				Items:     cleanedLineItems,
				VoiceName: line.VoiceName,
			})
		}

		if len(cleanedLines) == 0 {
			continue
		}

		cleanedItem := *item
		cleanedItem.Lines = cleanedLines
		result = append(result, &cleanedItem)
	}

	return result
}

// Discards subtitle items whose lines contain no visible text (e.g. items left with only styling tags or
// whitespace after prior cleaning).
func removeEmptyItems(items []*astisub.Item) []*astisub.Item {
	result := []*astisub.Item{}

	for _, item := range items {
		hasText := false
		for _, line := range item.Lines {
			for _, lineItem := range line.Items {
				text := stylingTagPattern.ReplaceAllString(lineItem.Text, "")
				if strings.TrimSpace(text) != "" {
					hasText = true
					break
				}
			}
			if hasText {
				break
			}
		}
		if hasText {
			result = append(result, item)
		}
	}

	return result
}

func writeToSRTFile(subs *astisub.Subtitles, path string) ([]backup, error) {
	backupFilePath, err := backupSubtitleFile(path)
	if err != nil {
		return nil, err
	}

	backups := []backup{
		{currentPath: backupFilePath, originalPath: path},
	}

	f, err := os.Create(path)
	if err != nil {
		return backups, fmt.Errorf("failed to create subtitle file: %w", err)
	}
	defer f.Close()

	err = subs.WriteToSRT(f)
	if err != nil {
		return backups, fmt.Errorf("failed to write subtitles to file: %w", err)
	}

	return backups, nil
}

func backupSubtitleFile(path string) (string, error) {
	dir := filepath.Dir(path)
	fileName := filepath.Base(path)
	backupFilePath := filepath.Join(dir, fmt.Sprintf("_%s.bak", fileName))

	if err := os.Rename(path, backupFilePath); err != nil {
		return "", fmt.Errorf("failed to backup subtitle file: %w", err)
	}

	return backupFilePath, nil
}
