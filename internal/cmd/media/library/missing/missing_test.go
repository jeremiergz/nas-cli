package missing

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jeremiergz/nas-cli/internal/service/tmdb"
)

func TestNewRegistersTypeSubcommandsAndAliases(t *testing.T) {
	cmd := New()
	tests := []struct {
		name    string
		aliases []string
	}{
		{name: "tvshows", aliases: []string{"tvshow", "tv", "t"}},
		{name: "animes", aliases: []string{"anime", "ani", "a"}},
	}
	for _, test := range tests {
		subcommand, _, err := cmd.Find([]string{test.name})
		if err != nil {
			t.Fatalf("Find(%q) error = %v", test.name, err)
		}
		if subcommand.Name() != test.name {
			t.Errorf("subcommand name = %q, want %q", subcommand.Name(), test.name)
		}
		if !slices.Equal(subcommand.Aliases, test.aliases) {
			t.Errorf("%s aliases = %v, want %v", test.name, subcommand.Aliases, test.aliases)
		}
	}
}

func TestRenderMissingBuildsTree(t *testing.T) {
	item := &show{name: "Example"}
	results := []showMissing{{
		show: item,
		episodes: []tmdb.Episode{
			{SeasonNumber: 1, EpisodeNumber: 2, Name: "Second"},
			{SeasonNumber: 1, EpisodeNumber: 4, Name: "Fourth"},
		},
	}}

	var output bytes.Buffer
	if err := renderMissing(&output, "tvshows", results, 2); err != nil {
		t.Fatalf("renderMissing() error = %v", err)
	}
	for _, want := range []string{
		"tvshows (2 missing episodes)",
		"Example (2 episodes)",
		"Season 1",
		"S01E02 Second",
		"S01E04 Fourth",
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("renderMissing() output missing %q:\n%s", want, output.String())
		}
	}
}

func TestCollectMissingCompactsParallelResults(t *testing.T) {
	results := []showMissing{
		{},
		{show: &show{name: "Missing show"}, episodes: []tmdb.Episode{{SeasonNumber: 1, EpisodeNumber: 1}}},
		{},
	}

	got, count := collectMissing(results, []bool{false, true, false})
	if len(got) != 1 || got[0].show.name != "Missing show" {
		t.Fatalf("collectMissing() results = %#v, want only missing show", got)
	}
	if count != 1 {
		t.Errorf("collectMissing() count = %d, want 1", count)
	}
}

func TestComparisonProgressIncludesCountDuringRateLimit(t *testing.T) {
	got := comparisonProgress(10, 500, time.Second)
	for _, want := range []string{"Compared 10/500 shows", "TMDB rate limited", "retrying in 1s"} {
		if !strings.Contains(got, want) {
			t.Errorf("comparisonProgress() = %q, want it to contain %q", got, want)
		}
	}
}

func TestPlexGUIDListUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{name: "string", data: `"tmdb://123"`, want: "tmdb://123"},
		{name: "object", data: `{"id":"tmdb://123"}`, want: "tmdb://123"},
		{name: "array", data: `[{"id":"tvdb://456"},{"id":"tmdb://123"}]`, want: "tvdb://456"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got plexGUIDList
			if err := json.Unmarshal([]byte(test.data), &got); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			if len(got) == 0 || got[0].ID != test.want {
				t.Errorf("GUIDs = %#v, want first ID %q", got, test.want)
			}
		})
	}
}

func TestTMDBIDFromGUIDs(t *testing.T) {
	got, err := tmdbIDFromGUIDs(plexGUIDList{
		{ID: "tvdb://123"},
		{ID: "tmdb://456"},
	})
	if err != nil {
		t.Fatalf("tmdbIDFromGUIDs() error = %v", err)
	}
	if got != "456" {
		t.Errorf("tmdbIDFromGUIDs() = %q, want %q", got, "456")
	}
}

func TestTMDBIDFromGUIDsRequiresTMDBID(t *testing.T) {
	if _, err := tmdbIDFromGUIDs(plexGUIDList{{ID: "tvdb://123"}}); err == nil {
		t.Fatal("tmdbIDFromGUIDs() error = nil, want missing TMDB ID error")
	}
}

func TestFindSectionID(t *testing.T) {
	got, err := findSectionID([]plexSection{
		{Key: "1", Title: "Movies"},
		{Key: "2", Title: "TV Shows"},
	}, "tvshows")
	if err != nil {
		t.Fatalf("findSectionID() error = %v", err)
	}
	if got != "2" {
		t.Errorf("findSectionID() = %q, want %q", got, "2")
	}
}

func TestMissingEpisodesOnlyReportsAiredEpisodesNotStored(t *testing.T) {
	now := time.Date(2026, time.October, 3, 12, 0, 0, 0, time.UTC)
	stored := map[int]map[int]struct{}{
		1: {1: {}},
	}
	catalog := []tmdb.Episode{
		{SeasonNumber: 0, EpisodeNumber: 1, AirDate: "2020-01-01", Name: "Special"},
		{SeasonNumber: 1, EpisodeNumber: 1, AirDate: "2026-01-01", Name: "Stored"},
		{SeasonNumber: 1, EpisodeNumber: 2, AirDate: "2026-10-03", Name: "Missing"},
		{SeasonNumber: 1, EpisodeNumber: 3, AirDate: "2026-10-04", Name: "Upcoming"},
		{SeasonNumber: 1, EpisodeNumber: 4, Name: "Unscheduled"},
	}

	got, err := missingEpisodes(stored, catalog, now)
	if err != nil {
		t.Fatalf("missingEpisodes() error = %v", err)
	}
	if len(got) != 1 || got[0].EpisodeNumber != 2 || got[0].Name != "Missing" {
		t.Errorf("missingEpisodes() = %#v, want only episode 2", got)
	}
}

func TestMissingEpisodesRejectsInvalidAirDate(t *testing.T) {
	_, err := missingEpisodes(nil, []tmdb.Episode{
		{SeasonNumber: 1, EpisodeNumber: 1, AirDate: "not-a-date"},
	}, time.Now())
	if err == nil {
		t.Fatal("missingEpisodes() error = nil, want invalid air date error")
	}
}
