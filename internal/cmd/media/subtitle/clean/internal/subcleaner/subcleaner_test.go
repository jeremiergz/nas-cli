package subcleaner

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/asticode/go-astisub"
	"github.com/jedib0t/go-pretty/v6/progress"
)

func TestMergeDuplicateTimestamps(t *testing.T) {
	items := []*astisub.Item{
		newItem(0, 1000, "Hello"),
		newItem(0, 1000, "World"),
		newItem(1000, 2000, "Separate"),
	}

	result := mergeDuplicateTimestamps(items)

	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	if len(result[0].Lines) != 2 {
		t.Fatalf("expected first item to have 2 lines, got %d", len(result[0].Lines))
	}
	texts := itemTexts(result)
	if texts[0] != "Hello" || texts[1] != "World" || texts[2] != "Separate" {
		t.Errorf("unexpected texts: %v", texts)
	}
}

func TestMergeDuplicateTimestamps_NoDuplicates(t *testing.T) {
	items := []*astisub.Item{
		newItem(0, 1000, "First"),
		newItem(1000, 2000, "Second"),
	}

	result := mergeDuplicateTimestamps(items)

	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
}

func TestRemoveSDH_Brackets(t *testing.T) {
	items := []*astisub.Item{
		newItem(0, 1000, "[gunshot] Run!"),
		newItem(1000, 2000, "(screaming) Help me!"),
	}

	result := removeSDH(items)
	texts := itemTexts(result)

	if len(texts) != 2 {
		t.Fatalf("expected 2 texts, got %d", len(texts))
	}
	if texts[0] != "Run!" {
		t.Errorf("expected 'Run!', got %q", texts[0])
	}
	if texts[1] != "Help me!" {
		t.Errorf("expected 'Help me!', got %q", texts[1])
	}
}

func TestRemoveSDH_FullyBracketed(t *testing.T) {
	items := []*astisub.Item{
		newItem(0, 1000, "[dramatic music]"),
		newItem(1000, 2000, "Normal text"),
	}

	result := removeSDH(items)

	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	texts := itemTexts(result)
	if texts[0] != "Normal text" {
		t.Errorf("expected 'Normal text', got %q", texts[0])
	}
}

func TestRemoveSDH_MusicSymbols(t *testing.T) {
	items := []*astisub.Item{
		newItem(0, 1000, "♪ La la la ♪"),
		newItem(1000, 2000, "♫♫"),
	}

	result := removeSDH(items)

	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	texts := itemTexts(result)
	if texts[0] != "La la la" {
		t.Errorf("expected 'La la la', got %q", texts[0])
	}
}

func TestRemoveSDH_SpeakerLabels(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"uppercase", "NARRATOR: Once upon a time", "Once upon a time"},
		{"mixed case", "Narrator: Once upon a time", "Once upon a time"},
		{"with number", "MAN 1: Hello there", "Hello there"},
		{"lowercase", "narrator: Once upon a time", "Once upon a time"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := []*astisub.Item{newItem(0, 1000, tt.input)}
			result := removeSDH(items)
			texts := itemTexts(result)
			if len(texts) != 1 || texts[0] != tt.expected {
				t.Errorf("expected %q, got %v", tt.expected, texts)
			}
		})
	}
}

func TestRemoveHTMLTags_StripsNonItalicTags(t *testing.T) {
	items := []*astisub.Item{
		newItem(0, 1000, `<font color="#ffff00">Yellow text</font>`),
		newItem(1000, 2000, "<b>Bold text</b>"),
		newItem(2000, 3000, "<u>Underlined</u>"),
	}

	result := removeHTMLTags(items)
	texts := itemTexts(result)

	if len(texts) != 3 {
		t.Fatalf("expected 3 texts, got %d", len(texts))
	}
	if texts[0] != "Yellow text" {
		t.Errorf("expected 'Yellow text', got %q", texts[0])
	}
	if texts[1] != "Bold text" {
		t.Errorf("expected 'Bold text', got %q", texts[1])
	}
	if texts[2] != "Underlined" {
		t.Errorf("expected 'Underlined', got %q", texts[2])
	}
}

func TestRemoveHTMLTags_PreservesItalic(t *testing.T) {
	items := []*astisub.Item{
		newItem(0, 1000, "<i>Italic text</i>"),
		newItem(1000, 2000, `<font color="red"><i>Styled italic</i></font>`),
	}

	result := removeHTMLTags(items)
	texts := itemTexts(result)

	if len(texts) != 2 {
		t.Fatalf("expected 2 texts, got %d", len(texts))
	}
	if texts[0] != "<i>Italic text</i>" {
		t.Errorf("expected '<i>Italic text</i>', got %q", texts[0])
	}
	if texts[1] != "<i>Styled italic</i>" {
		t.Errorf("expected '<i>Styled italic</i>', got %q", texts[1])
	}
}

func TestRemoveHTMLTags_EmptyAfterStrip(t *testing.T) {
	items := []*astisub.Item{
		newItem(0, 1000, "<font> </font>"),
		newItem(1000, 2000, "Keep this"),
	}

	result := removeHTMLTags(items)

	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	texts := itemTexts(result)
	if texts[0] != "Keep this" {
		t.Errorf("expected 'Keep this', got %q", texts[0])
	}
}

func TestRemoveEmptyItems_StylingTagOnly(t *testing.T) {
	items := []*astisub.Item{
		newItem(0, 1000, `{\an8}`),
		newItem(1000, 2000, `{\an8}Hello`),
		newItem(2000, 3000, "   "),
	}

	result := removeEmptyItems(items)

	if len(result) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result))
	}
	texts := itemTexts(result)
	if texts[0] != `{\an8}Hello` {
		t.Errorf(`expected '{\an8}Hello', got %q`, texts[0])
	}
}

func TestFullPipeline(t *testing.T) {
	items := []*astisub.Item{
		newItem(0, 1000, "[thunder rumbling]"),
		newItem(1000, 2000, "NARRATOR: It was a dark night."),
		newItem(1000, 2000, `{\an8}(wind howling)`),
		newItem(2000, 3000, `<font color="#ffff00">♪ Theme song ♪</font>`),
		newItem(3000, 4000, "<i>Whispered words</i>"),
		newItem(4000, 5000, "Normal dialogue here."),
		newItem(4000, 5000, "Duplicate timestamp line."),
	}

	cleanedItems := items
	for _, cleanup := range cleanupPipeline {
		cleanedItems = cleanup(cleanedItems)
	}

	texts := itemTexts(cleanedItems)

	// [thunder rumbling] -> removed entirely (SDH)
	// NARRATOR: It was a dark night. -> "It was a dark night." (speaker label)
	//   merged with {\an8}(wind howling) -> SDH removes parens, {\an8} kept as line in same item
	// <font ...>♪ Theme song ♪</font> -> music symbols removed, font tags removed -> "Theme song"
	// <i>Whispered words</i> -> preserved
	// Normal dialogue here. + Duplicate timestamp line. -> merged (same timestamps)

	expected := []string{
		"It was a dark night.",
		`{\an8}`,
		"Theme song",
		"<i>Whispered words</i>",
		"Normal dialogue here.",
		"Duplicate timestamp line.",
	}

	if len(texts) != len(expected) {
		t.Fatalf("expected %d texts, got %d: %v", len(expected), len(texts), texts)
	}
	for i, exp := range expected {
		if texts[i] != exp {
			t.Errorf("text[%d]: expected %q, got %q", i, exp, texts[i])
		}
	}
}

func TestRun_IntegrationWithSRTFile(t *testing.T) {
	srtPath := copyTestdataFile(t)

	proc := New(srtPath, false)

	var buf bytes.Buffer
	proc.SetTracker(newTestTracker()).SetOutput(&buf)

	if err := proc.Run(context.Background()); err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	resultBytes, err := os.ReadFile(srtPath)
	if err != nil {
		t.Fatalf("failed to read result: %v", err)
	}
	result := string(resultBytes)

	// Should be cleaned.
	shouldNotContain := []struct {
		text   string
		reason string
	}{
		{"[thunder rumbling]", "SDH bracket annotation should have been removed"},
		{"(dramatic music intensifies)", "SDH parenthesized annotation should have been removed"},
		{"[door creaks open]", "SDH bracket annotation should have been removed"},
		{"NARRATOR:", "uppercase speaker label should have been removed"},
		{"Narrator:", "mixed-case speaker label should have been removed"},
		{"MAN 1:", "speaker label with number should have been removed"},
		{"WOMAN:", "speaker label should have been removed"},
		{"♪", "music symbols should have been removed"},
		{"♫", "music symbols should have been removed"},
		{"<font", "<font> tag should have been removed"},
		{"<b>", "<b> tag should have been removed"},
		{"</b>", "</b> tag should have been removed"},
		{"<u>", "<u> tag should have been removed"},
		{"</u>", "</u> tag should have been removed"},
		{"<span", "<span> tag should have been removed"},
	}
	for _, tc := range shouldNotContain {
		if strings.Contains(result, tc.text) {
			t.Errorf("%s (found %q in output)", tc.reason, tc.text)
		}
	}

	// Should be kept.
	shouldContain := []struct {
		text   string
		reason string
	}{
		{"It was a dark and stormy night.", "dialogue after speaker label removal should be preserved"},
		{"The wind howled through the trees.", "dialogue after speaker label removal should be preserved"},
		{"Did you hear that?", "dialogue after speaker label removal should be preserved"},
		{"What was that?", "dialogue after SDH + speaker label removal should be preserved"},
		{"This is yellow text", "text inside <font> should be preserved"},
		{"Bold text here", "text inside <b> should be preserved"},
		{"Underlined text", "text inside <u> should be preserved"},
		{"<i>Styled italic text</i>", "<i> tags should be preserved"},
		{"<i>Regular italic text</i>", "<i> tags should be preserved"},
		{"Text with positioning tag", "text with {\\an8} prefix should be preserved"},
		{"Be very quiet.", "dialogue after SDH removal should be preserved"},
		{"Normal dialogue line.", "normal dialogue should be preserved"},
		{"Duplicate timestamp line.", "duplicate timestamp text should be preserved (merged)"},
		{"I can't believe it.", "dialogue after SDH removal should be preserved"},
		{"Span styled text", "text inside <span> should be preserved"},
		{"Run!", "dialogue after combined SDH removal should be preserved"},
		{"Big bold text", "text inside nested HTML tags should be preserved"},
	}
	for _, tc := range shouldContain {
		if !strings.Contains(result, tc.text) {
			t.Errorf("%s (expected %q in output)", tc.reason, tc.text)
		}
	}

	// Empty items should be gone: {\an8} alone, whitespace-only, music-only.
	// Verify no empty subtitle lines remain.
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == `{\an8}` {
			t.Errorf("line %d: empty {\\an8} item should have been removed", i+1)
		}
	}

	// Backup should have been removed (keepOriginal=false).
	backupPath := filepath.Join(filepath.Dir(srtPath), "_dirty.srt.bak")
	if _, err := os.Stat(backupPath); err == nil {
		t.Error("backup file should have been removed when keepOriginal=false")
	}
}

func TestRun_KeepOriginal(t *testing.T) {
	srtPath := copyTestdataFile(t)

	proc := New(srtPath, true)

	var buf bytes.Buffer
	proc.SetTracker(newTestTracker()).SetOutput(&buf)

	if err := proc.Run(context.Background()); err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	// Backup should still exist.
	backupPath := filepath.Join(filepath.Dir(srtPath), "_dirty.srt.bak")
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("backup file should exist when keepOriginal=true")
	}
}

func TestRun_NoTracker(t *testing.T) {
	proc := New("/tmp/nonexistent.srt", false)
	err := proc.Run(context.Background())
	if err == nil {
		t.Fatal("expected error when tracker is nil")
	}
	if err.Error() != "required tracker is not set" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRun_FileOpenError(t *testing.T) {
	proc := New("/nonexistent/path/file.srt", false)

	var buf bytes.Buffer
	proc.SetTracker(newTestTracker()).SetOutput(&buf)

	err := proc.Run(context.Background())
	if err == nil {
		t.Fatal("expected error when file does not exist")
	}
}

func TestRun_InvalidSRTFile(t *testing.T) {
	dir := t.TempDir()
	srtPath := filepath.Join(dir, "bad.srt")

	if err := os.WriteFile(srtPath, []byte("not a valid srt file\x00\x01\x02"), 0o644); err != nil {
		t.Fatalf("failed to write bad SRT: %v", err)
	}

	proc := New(srtPath, false)

	var buf bytes.Buffer
	proc.SetTracker(newTestTracker()).SetOutput(&buf)

	err := proc.Run(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid SRT content")
	}
}

func TestBackupSubtitleFile_Error(t *testing.T) {
	_, err := backupSubtitleFile("/nonexistent/path/file.srt")
	if err == nil {
		t.Fatal("expected error when file does not exist")
	}
	if !strings.Contains(err.Error(), "failed to backup subtitle file") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWriteToSRTFile_BackupError(t *testing.T) {
	subs := astisub.NewSubtitles()
	_, err := writeToSRTFile(subs, "/nonexistent/path/file.srt")
	if err == nil {
		t.Fatal("expected error when backup fails")
	}
}

func TestWriteToSRTFile_CreateError(t *testing.T) {
	// Create a file, then make the directory read-only so os.Create fails after backup.
	dir := t.TempDir()
	srtPath := filepath.Join(dir, "test.srt")

	if err := os.WriteFile(srtPath, []byte("1\n00:00:00,000 --> 00:00:01,000\nHi\n\n"), 0o644); err != nil {
		t.Fatalf("failed to write SRT: %v", err)
	}

	// Backup will succeed (rename), then make dir read-only so Create fails.
	backupPath := filepath.Join(dir, "_test.srt.bak")

	// Do the backup manually first, then make dir unwritable.
	if err := os.Rename(srtPath, backupPath); err != nil {
		t.Fatalf("failed to rename: %v", err)
	}

	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	// Attempt to write - backup file exists but dir isn't writable, so rename in backupSubtitleFile will fail.
	subs := astisub.NewSubtitles()
	_, err := writeToSRTFile(subs, srtPath)
	if err == nil {
		t.Fatal("expected error when directory is not writable")
	}
}

// Helper to build a subtitle item with a single line of text.
func newItem(startMs, endMs int, texts ...string) *astisub.Item {
	var lines []astisub.Line
	for _, t := range texts {
		lines = append(lines, astisub.Line{
			Items: []astisub.LineItem{{Text: t}},
		})
	}
	return &astisub.Item{
		StartAt: time.Duration(startMs) * time.Millisecond,
		EndAt:   time.Duration(endMs) * time.Millisecond,
		Lines:   lines,
	}
}

func itemTexts(items []*astisub.Item) []string {
	var out []string
	for _, item := range items {
		for _, line := range item.Lines {
			for _, li := range line.Items {
				out = append(out, li.Text)
			}
		}
	}
	return out
}

func copyTestdataFile(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile("testdata/dirty.srt")
	if err != nil {
		t.Fatalf("failed to read testdata/dirty.srt: %v", err)
	}
	dir := t.TempDir()
	dst := filepath.Join(dir, "dirty.srt")
	if err := os.WriteFile(dst, src, 0o644); err != nil {
		t.Fatalf("failed to write temp SRT: %v", err)
	}
	return dst
}

func newTestTracker() *progress.Tracker {
	tracker := &progress.Tracker{
		Message: "test",
		Total:   1,
	}
	pw := progress.NewWriter()
	pw.AppendTracker(tracker)
	return tracker
}
