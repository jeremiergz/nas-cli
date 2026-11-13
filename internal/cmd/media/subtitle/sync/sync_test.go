package sync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jeremiergz/nas-cli/internal/media"
)

func TestLanguageForSubtitle(t *testing.T) {
	tests := []struct {
		name     string
		subtitle string
		override string
		want     string
	}{
		{name: "infer French", subtitle: "Movie.fra.srt", want: "fre"},
		{name: "infer legacy French", subtitle: "Movie.fre.srt", want: "fre"},
		{name: "infer forced subtitle", subtitle: "Movie.fra.forced.srt", want: "fre"},
		{name: "ignore non-language suffix", subtitle: "Movie.1080p.srt", want: ""},
		{name: "fallback without suffix", subtitle: "Movie.srt", want: ""},
		{name: "explicit override", subtitle: "Movie.fra.srt", override: "eng", want: "eng"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := languageForSubtitle(test.subtitle, test.override); got != test.want {
				t.Errorf("languageForSubtitle(%q, %q) = %q, want %q", test.subtitle, test.override, got, test.want)
			}
		})
	}
}

func TestOutputSubtitleName(t *testing.T) {
	tests := []struct {
		video    string
		language string
		forced   bool
		want     string
	}{
		{video: "Movie.mkv", language: "fre", want: "Movie.fre.srt"},
		{video: "Movie.mkv", language: "fre", forced: true, want: "Movie.fre.forced.srt"},
		{video: "Movie.mkv", want: "Movie.srt"},
	}
	for _, test := range tests {
		if got := outputSubtitleName(test.video, test.language, test.forced); got != test.want {
			t.Errorf("outputSubtitleName(%q, %q, %t) = %q, want %q", test.video, test.language, test.forced, got, test.want)
		}
	}
}

func TestMatchingSubtitles(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		"Movie A.mkv",
		"Movie A.eng.srt",
		"Movie A.fra.srt",
		"Movie A.fre.forced.srt",
		"Movie B.mkv",
		"Movie B.spa.srt",
		"Unrelated.eng.srt",
	}
	for _, file := range files {
		if err := os.WriteFile(filepath.Join(dir, file), nil, 0600); err != nil {
			t.Fatalf("create %s: %v", file, err)
		}
	}

	videos, err := media.Files(dir, []string{"mkv"}, false)
	if err != nil {
		t.Fatal(err)
	}
	jobs := matchingSubtitles(videos, "srt")

	want := []syncJob{
		{video: "Movie A.mkv", subtitle: "Movie A.eng.srt"},
		{video: "Movie A.mkv", subtitle: "Movie A.fra.srt"},
		{video: "Movie A.mkv", subtitle: "Movie A.fre.forced.srt", forced: true},
		{video: "Movie B.mkv", subtitle: "Movie B.spa.srt"},
	}
	if len(jobs) != len(want) {
		t.Fatalf("matchingSubtitles() returned %d jobs, want %d: %+v", len(jobs), len(want), jobs)
	}
	for i := range want {
		if jobs[i] != want[i] {
			t.Errorf("matchingSubtitles()[%d] = %+v, want %+v", i, jobs[i], want[i])
		}
	}
}
