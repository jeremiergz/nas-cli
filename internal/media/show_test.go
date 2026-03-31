package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseShows(t *testing.T) {
	dir := t.TempDir()

	// Parser expects filenames like "Show.Name.S01E01.Resolution.Container".
	filenames := []string{
		"Breaking.Bad.S01E01.720p.mkv",
		"Breaking.Bad.S01E02.720p.mkv",
		"Breaking.Bad.S02E01.720p.mkv",
	}
	for _, name := range filenames {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", name, err)
		}
	}

	shows, err := ParseShows(dir, []string{"mkv"}, false)
	if err != nil {
		t.Fatalf("ParseShows() error: %v", err)
	}

	if len(shows) != 1 {
		t.Fatalf("expected 1 show, got %d", len(shows))
	}

	show := shows[0]
	if show.Name() != "Breaking Bad" {
		t.Errorf("show.Name() = %q, want %q", show.Name(), "Breaking Bad")
	}
	if show.SeasonsCount() != 2 {
		t.Errorf("SeasonsCount() = %d, want %d", show.SeasonsCount(), 2)
	}
	if show.EpisodesCount() != 3 {
		t.Errorf("EpisodesCount() = %d, want %d", show.EpisodesCount(), 3)
	}

	// Seasons should be sorted by index.
	seasons := show.Seasons()
	if seasons[0].Index() != 1 {
		t.Errorf("seasons[0].Index() = %d, want 1", seasons[0].Index())
	}
	if seasons[1].Index() != 2 {
		t.Errorf("seasons[1].Index() = %d, want 2", seasons[1].Index())
	}

	// Season 1 should have 2 episodes sorted by index.
	s1Episodes := seasons[0].Episodes()
	if len(s1Episodes) != 2 {
		t.Fatalf("season 1 expected 2 episodes, got %d", len(s1Episodes))
	}
	if s1Episodes[0].Index() != 1 {
		t.Errorf("s1e1.Index() = %d, want 1", s1Episodes[0].Index())
	}
	if s1Episodes[1].Index() != 2 {
		t.Errorf("s1e2.Index() = %d, want 2", s1Episodes[1].Index())
	}
}

func TestListShows(t *testing.T) {
	type expectedEpisode struct {
		fileName      string
		showName      string
		seasonNumber  int
		episodeNumber int
	}

	tests := []struct {
		name       string
		files      []string
		extensions []string
		expected   []expectedEpisode
	}{
		{
			name: "multiple shows with multiple seasons",
			files: []string{
				"One Piece - S01E0001.mkv",
				"One Piece - S01E0002.mkv",
				"One Piece - S01E1100.mkv",
				"One Piece - S01E1201.mkv",
				"The Office - S01E01.mkv",
				"The Office - S01E02.mkv",
				"The Office - S02E01.mkv",
				"The Office - S02E05.mkv",
				"Friends - S01E01.mkv",
				"Friends - S03E10.mkv",
			},
			extensions: []string{"mkv"},
			expected: []expectedEpisode{
				{"One Piece - S01E01.mkv", "One Piece", 1, 1},
				{"One Piece - S01E02.mkv", "One Piece", 1, 2},
				{"One Piece - S01E1100.mkv", "One Piece", 1, 1100},
				{"One Piece - S01E1201.mkv", "One Piece", 1, 1201},
				{"The Office - S01E01.mkv", "The Office", 1, 1},
				{"The Office - S01E02.mkv", "The Office", 1, 2},
				{"The Office - S02E01.mkv", "The Office", 2, 1},
				{"The Office - S02E05.mkv", "The Office", 2, 5},
				{"Friends - S01E01.mkv", "Friends", 1, 1},
				{"Friends - S03E10.mkv", "Friends", 3, 10},
			},
		},
		{
			name: "filter by extension",
			files: []string{
				"Show - S01E01.mkv",
				"Show - S01E02.avi",
				"Show - S01E03.mkv",
			},
			extensions: []string{"mkv"},
			expected: []expectedEpisode{
				{"Show - S01E01.mkv", "Show", 1, 1},
				{"Show - S01E03.mkv", "Show", 1, 3},
			},
		},
		{
			name:       "empty directory",
			files:      []string{},
			extensions: []string{"mkv"},
			expected:   []expectedEpisode{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			for _, name := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, name), nil, 0644); err != nil {
					t.Fatalf("failed to create file %s: %v", name, err)
				}
			}

			shows, err := ListShows(dir, tt.extensions, false)
			if err != nil {
				t.Fatalf("ListShows() error: %v", err)
			}

			// Collect all episodes from the result.
			type actualEpisode struct {
				fileName      string
				showName      string
				seasonNumber  int
				episodeNumber int
			}
			var got []actualEpisode
			for _, show := range shows {
				for _, season := range show.Seasons() {
					for _, ep := range season.Episodes() {
						got = append(got, actualEpisode{
							fileName:      ep.FullName(),
							showName:      show.Name(),
							seasonNumber:  season.Index(),
							episodeNumber: ep.Index(),
						})
					}
				}
			}

			if len(got) != len(tt.expected) {
				t.Fatalf("expected %d episodes, got %d", len(tt.expected), len(got))
			}

			// Build a lookup from the actual episodes for order-independent matching.
			lookup := map[string]actualEpisode{}
			for _, ep := range got {
				lookup[ep.fileName] = ep
			}

			for _, want := range tt.expected {
				actual, ok := lookup[want.fileName]
				if !ok {
					t.Errorf("expected episode %q not found", want.fileName)
					continue
				}
				if actual.showName != want.showName {
					t.Errorf("%s: showName = %q, want %q", want.fileName, actual.showName, want.showName)
				}
				if actual.seasonNumber != want.seasonNumber {
					t.Errorf("%s: seasonNumber = %d, want %d", want.fileName, actual.seasonNumber, want.seasonNumber)
				}
				if actual.episodeNumber != want.episodeNumber {
					t.Errorf("%s: episodeNumber = %d, want %d", want.fileName, actual.episodeNumber, want.episodeNumber)
				}
			}
		})
	}
}

func newTestEpisode() *Episode {
	show := &Show{name: "Breaking Bad"}
	season := &Season{name: "Season 1", index: 1, show: show}
	f, _ := newFile("Breaking.Bad.S01E01.720p.mkv", "mkv", "/tmp/Breaking.Bad.S01E01.720p.mkv")
	return &Episode{file: f, index: 1, season: season}
}

func TestEpisodeIndex(t *testing.T) {
	ep := newTestEpisode()
	if ep.Index() != 1 {
		t.Errorf("Index() = %d, want 1", ep.Index())
	}
}

func TestEpisodeSeason(t *testing.T) {
	ep := newTestEpisode()
	if ep.Season().Name() != "Season 1" {
		t.Errorf("Season().Name() = %q, want %q", ep.Season().Name(), "Season 1")
	}
}

func TestEpisodeName(t *testing.T) {
	ep := newTestEpisode()
	if ep.Name() != "Breaking Bad - S01E01" {
		t.Errorf("Name() = %q, want %q", ep.Name(), "Breaking Bad - S01E01")
	}
}

func TestEpisodeFullName(t *testing.T) {
	ep := newTestEpisode()
	if ep.FullName() != "Breaking Bad - S01E01.mkv" {
		t.Errorf("FullName() = %q, want %q", ep.FullName(), "Breaking Bad - S01E01.mkv")
	}
}

func TestEpisodePosterImageFilePath(t *testing.T) {
	ep := newTestEpisode()
	if ep.PosterImageFilePath() != nil {
		t.Error("PosterImageFilePath() should return nil")
	}
}

func TestSeasonName(t *testing.T) {
	show := &Show{name: "Test Show"}
	season := &Season{name: "Season 2", index: 2, show: show, episodes: []*Episode{}}
	if season.Name() != "Season 2" {
		t.Errorf("Name() = %q, want %q", season.Name(), "Season 2")
	}
}

func TestSeasonIndex(t *testing.T) {
	show := &Show{name: "Test Show"}
	season := &Season{name: "Season 2", index: 2, show: show}
	if season.Index() != 2 {
		t.Errorf("Index() = %d, want 2", season.Index())
	}
}

func TestSeasonShow(t *testing.T) {
	show := &Show{name: "Test Show"}
	season := &Season{name: "Season 2", index: 2, show: show}
	if season.Show() != show {
		t.Error("Show() did not return expected show")
	}
}

func TestSeasonEpisodes(t *testing.T) {
	show := &Show{name: "Test Show"}
	season := &Season{name: "Season 2", index: 2, show: show, episodes: []*Episode{}}
	if len(season.Episodes()) != 0 {
		t.Errorf("Episodes() length = %d, want 0", len(season.Episodes()))
	}
}

func TestShowSetName(t *testing.T) {
	show := &Show{name: "Old Name"}
	show.SetName("New Name")
	if show.Name() != "New Name" {
		t.Errorf("after SetName, Name() = %q, want %q", show.Name(), "New Name")
	}
}

func TestParseShows_MultipleShows(t *testing.T) {
	dir := t.TempDir()

	filenames := []string{
		"Show.A.S01E01.720p.mkv",
		"Show.B.S01E01.720p.mkv",
		"Show.B.S01E02.720p.mkv",
	}
	for _, name := range filenames {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0644); err != nil {
			t.Fatalf("failed to create file %s: %v", name, err)
		}
	}

	shows, err := ParseShows(dir, []string{"mkv"}, false)
	if err != nil {
		t.Fatalf("ParseShows() error: %v", err)
	}

	if len(shows) != 2 {
		t.Fatalf("expected 2 shows, got %d", len(shows))
	}
}
