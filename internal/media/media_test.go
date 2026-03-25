package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFiles(t *testing.T) {
	dir := t.TempDir()

	names := []string{"video1.mkv", "video2.mkv", "video3.mkv"}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0644); err != nil {
			t.Fatalf("failed to create temp file %s: %v", name, err)
		}
	}

	// Also create a non-matching file to verify filtering.
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), nil, 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	files, err := Files(dir, []string{"mkv"}, false)
	if err != nil {
		t.Fatalf("Files() returned error: %v", err)
	}

	if len(files) != len(names) {
		t.Fatalf("expected %d files, got %d", len(names), len(files))
	}

	for i, f := range files {
		if f.Basename() != names[i] {
			t.Errorf("expected basename %q, got %q", names[i], f.Basename())
		}
		if f.Extension() != "mkv" {
			t.Errorf("expected extension %q, got %q", "mkv", f.Extension())
		}
		if f.FilePath() != filepath.Join(dir, names[i]) {
			t.Errorf("expected file path %q, got %q", filepath.Join(dir, names[i]), f.FilePath())
		}
	}
}

func TestSubtitles(t *testing.T) {
	tests := []struct {
		name          string
		videoFile     string
		neighborFiles []string
		filterLangs   []string
		wantSubtitles map[string]string
	}{
		{
			name:      "finds matching subtitles",
			videoFile: "movie.mkv",
			neighborFiles: []string{
				"movie.eng.srt",
				"movie.spa.srt",
			},
			wantSubtitles: map[string]string{
				"eng": "movie.eng.srt",
				"spa": "movie.spa.srt",
			},
		},
		{
			name:          "no subtitles present",
			videoFile:     "movie.mkv",
			wantSubtitles: map[string]string{},
		},
		{
			name:      "ignores non-srt files",
			videoFile: "movie.mkv",
			neighborFiles: []string{
				"movie.eng.sub",
				"movie.eng.txt",
			},
			wantSubtitles: map[string]string{},
		},
		{
			name:      "ignores subtitles for different video",
			videoFile: "movie.mkv",
			neighborFiles: []string{
				"other.eng.srt",
			},
			wantSubtitles: map[string]string{},
		},
		{
			name:      "filters by language",
			videoFile: "movie.mkv",
			neighborFiles: []string{
				"movie.eng.srt",
				"movie.spa.srt",
				"movie.fra.srt",
			},
			filterLangs: []string{"eng", "fra"},
			wantSubtitles: map[string]string{
				"eng": "movie.eng.srt",
				"fra": "movie.fra.srt",
			},
		},
		{
			name:      "should ignore diacritics and case when matching subtitles",
			videoFile: "movie.mkv",
			neighborFiles: []string{
				"móvíé.ENG.srt",
				"MÓVÍÉ.spa.srt",
			},
			wantSubtitles: map[string]string{
				"eng": "móvíé.ENG.srt",
				"spa": "MÓVÍÉ.spa.srt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			videoPath := filepath.Join(dir, tt.videoFile)
			if err := os.WriteFile(videoPath, nil, 0644); err != nil {
				t.Fatalf("failed to create video file: %v", err)
			}
			for _, name := range tt.neighborFiles {
				if err := os.WriteFile(filepath.Join(dir, name), nil, 0644); err != nil {
					t.Fatalf("failed to create file %s: %v", name, err)
				}
			}

			basename := filepath.Base(tt.videoFile)
			ext := filepath.Ext(basename)[1:]
			f, err := newFile(basename, ext, videoPath)
			if err != nil {
				t.Fatalf("newFile() returned error: %v", err)
			}

			got := f.Subtitles(tt.filterLangs...)
			if len(got) != len(tt.wantSubtitles) {
				t.Fatalf("expected %d subtitles, got %d: %v", len(tt.wantSubtitles), len(got), got)
			}
			for lang, wantFile := range tt.wantSubtitles {
				if got[lang] != wantFile {
					t.Errorf("lang %q: expected %q, got %q", lang, wantFile, got[lang])
				}
			}
		})
	}
}

func TestFiles_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	files, err := Files(dir, []string{"mkv"}, false)
	if err != nil {
		t.Fatalf("Files() returned error: %v", err)
	}

	if len(files) != 0 {
		t.Fatalf("expected 0 files, got %d", len(files))
	}
}
