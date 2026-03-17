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
	replaceHTLMEntities,
	removeSDH,
	removeHTMLTags,
	removeEmptyItems,
	fixDashSpacing,
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
	colonPrefixPattern = regexp.MustCompile(`^[A-Z][A-Z0-9 ]*:\s*`)

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
// Handles multi-line SDH annotations where an opening delimiter appears on one line
// and the closing delimiter on a subsequent line.
func removeSDH(items []*astisub.Item) []*astisub.Item {
	result := []*astisub.Item{}

	for _, item := range items {
		var cleanedLines []astisub.Line
		inParenSDH := false
		inBracketSDH := false

		for _, line := range item.Lines {
			var cleanedLineItems []astisub.LineItem

			for _, lineItem := range line.Items {
				text := lineItem.Text

				// Continue removing multi-line parenthesized SDH from a previous line.
				if inParenSDH {
					if idx := strings.Index(text, ")"); idx >= 0 {
						text = text[idx+1:]
						inParenSDH = false
					} else {
						continue
					}
				}

				// Continue removing multi-line bracketed SDH from a previous line.
				if inBracketSDH {
					if idx := strings.Index(text, "]"); idx >= 0 {
						text = text[idx+1:]
						inBracketSDH = false
					} else {
						continue
					}
				}

				// Remove single-line bracketed and parenthesized SDH annotations.
				text = sdhPattern.ReplaceAllString(text, "")

				// Detect unmatched opening parenthesis (start of multi-line SDH).
				if idx := strings.LastIndex(text, "("); idx >= 0 {
					text = text[:idx]
					inParenSDH = true
				}

				// Detect unmatched opening bracket (start of multi-line SDH).
				if idx := strings.LastIndex(text, "["); idx >= 0 {
					text = text[:idx]
					inBracketSDH = true
				}

				// Remove music symbols.
				text = musicPattern.ReplaceAllString(text, "")

				// Remove speaker labels (e.g. "NARRATOR:", "MAN 1:").
				text = colonPrefixPattern.ReplaceAllString(text, "")

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

// Discards subtitle lines that are empty or contain only a lone dash, then discards items whose remaining lines
// contain no visible text (e.g. items left with only styling tags or whitespace after prior cleaning).
func removeEmptyItems(items []*astisub.Item) []*astisub.Item {
	result := []*astisub.Item{}

	for _, item := range items {
		// Filter out lines that are empty or consist of only a dash (possibly wrapped in styling tags).
		// Lines containing only styling tags (e.g. {\an8}) without any dash are preserved.
		var cleanedLines []astisub.Line
		for _, line := range item.Lines {
			keepLine := true
			for _, lineItem := range line.Items {
				stripped := stylingTagPattern.ReplaceAllString(lineItem.Text, "")
				stripped = strings.TrimSpace(stripped)
				if stripped == "-" {
					keepLine = false
					break
				}
			}
			if keepLine {
				cleanedLines = append(cleanedLines, line)
			}
		}

		if len(cleanedLines) == 0 {
			continue
		}

		// Check whether any remaining line has visible text after stripping styling tags.
		hasText := false
		for _, line := range cleanedLines {
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

		if !hasText {
			continue
		}

		cleanedItem := *item
		cleanedItem.Lines = cleanedLines
		result = append(result, &cleanedItem)
	}

	return result
}

// htmlEntityPattern matches HTML entities not already decoded by the SRT reader
// (go-astisub decodes &amp;, &lt;, and &nbsp; on read).
var htmlEntityPattern = regexp.MustCompile(`&(nbsp|quot|apos|copy|reg|trade);`)

var htmlEntityReplacements = map[string]string{
	"nbsp":  " ",
	"quot":  `"`,
	"apos":  "'",
	"copy":  "©",
	"reg":   "®",
	"trade": "™",
}

// Handles Unicode characters that the SRT reader produces from decoded HTML entities (e.g. &nbsp; → U+00A0).
var htmlEntityReplacer = strings.NewReplacer(
	"\u00A0", " ", // non-breaking space → regular space
)

// Replaces HTML entities and their decoded Unicode equivalents with plain-text counterparts.
// The SRT reader decodes &amp;, &lt;, and &nbsp; into their Unicode forms;
// this function normalizes those (e.g. U+00A0 → space) and strips remaining
// entity strings (&gt;, &quot;, &apos;, &lrm;, &rlm;).
// Lines that become empty after replacement are discarded.
func replaceHTLMEntities(items []*astisub.Item) []*astisub.Item {
	result := []*astisub.Item{}

	for _, item := range items {
		var cleanedLines []astisub.Line

		for _, line := range item.Lines {
			var cleanedLineItems []astisub.LineItem

			for _, lineItem := range line.Items {
				// Replace decoded Unicode characters (e.g. NBSP).
				text := htmlEntityReplacer.Replace(lineItem.Text)

				// Replace remaining entity strings the SRT reader didn't decode.
				text = htmlEntityPattern.ReplaceAllStringFunc(text, func(match string) string {
					key := match[1 : len(match)-1] // Strip & and ;.
					if repl, ok := htmlEntityReplacements[key]; ok {
						return repl
					}
					return match
				})

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

// dashNoSpacePattern matches a leading dash immediately followed by a non-space character.
var dashNoSpacePattern = regexp.MustCompile(`^- *(?P<rest>\S)`)

// Ensures a space exists between a leading dash and the following character (e.g. "-Hey" becomes "- Hey").
func fixDashSpacing(items []*astisub.Item) []*astisub.Item {
	for _, item := range items {
		for i, line := range item.Lines {
			for j, lineItem := range line.Items {
				if strings.HasPrefix(lineItem.Text, "-") {
					item.Lines[i].Items[j].Text = dashNoSpacePattern.ReplaceAllString(lineItem.Text, "- ${rest}")
				}
			}
		}
	}
	return items
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
