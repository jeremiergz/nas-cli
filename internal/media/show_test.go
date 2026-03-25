package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListShows(t *testing.T) {
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

	shows, err := ListShows(dir, []string{"mkv"}, false, "srt", nil, false)
	if err != nil {
		t.Fatalf("ListShows() error: %v", err)
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

func TestListShows_MultipleShows(t *testing.T) {
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

	shows, err := ListShows(dir, []string{"mkv"}, false, "srt", nil, false)
	if err != nil {
		t.Fatalf("ListShows() error: %v", err)
	}

	if len(shows) != 2 {
		t.Fatalf("expected 2 shows, got %d", len(shows))
	}
}
