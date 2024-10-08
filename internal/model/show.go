package model

import (
	"fmt"
	"path"
	"regexp"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/jeremiergz/nas-cli/internal/model/internal/parser"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	_ MediaFile = (*Episode)(nil)

	episodeNameCaser = cases.Title(language.Und)
	showFmtRegexp    = regexp.MustCompile(`(^.+)(\s-\s)S\d+E\d+\.(.+)$`)
)

// Holds information about a show such as its name and seasons.
type Show struct {
	name          string
	seasons       []*Season
	seasonsCount  int
	episodesCount int
}

func Shows(wd string, extensions []string, subtitleExtension string, subtitleLangs []string, anyFiles bool) ([]*Show, error) {
	var selectedRegexp *regexp.Regexp
	if !anyFiles {
		selectedRegexp = showFmtRegexp
	}

	toProcess := fsutil.List(wd, extensions, selectedRegexp)
	shows := []*Show{}
	for _, basename := range toProcess {
		e, err := parser.Parse(basename)

		e.Title = episodeNameCaser.String(e.Title)
		if err == nil {
			var show *Show
			showIndex := findShowIndex(e.Title, shows)
			if showIndex == -1 {
				show = &Show{
					name:    e.Title,
					seasons: []*Season{},
				}
			} else {
				show = shows[showIndex]
			}
			seasonName := fmt.Sprintf("Season %d", e.Season)
			seasonIndex := findShowSeasonIndex(seasonName, show.Seasons())
			episode := Episode{
				file: file{
					basename:  basename,
					extension: e.Container,
					filePath:  path.Join(wd, basename),
				},
				index: e.Episode,
			}

			var season *Season
			if seasonIndex == -1 {
				season = &Season{
					episodes: []*Episode{},
					index:    e.Season,
					name:     seasonName,
					show:     show,
				}
				episode.season = season
				season.episodes = append(season.episodes, &episode)
				show.seasons = append(show.seasons, season)
			} else {
				season := show.seasons[seasonIndex]
				episode.season = season
				season.episodes = append(season.episodes, &episode)
			}
			if showIndex == -1 {
				shows = append(shows, show)
			}
		} else {
			return nil, err
		}
	}

	for _, show := range shows {
		show.seasonsCount = len(show.seasons)
		for _, season := range show.seasons {
			show.episodesCount += len(season.episodes)
		}
	}

	return shows, nil
}

func (s *Show) Name() string {
	return s.name
}

func (s *Show) SetName(name string) {
	s.name = name
}

func (s *Show) Seasons() []*Season {
	return s.seasons
}

func (s *Show) SeasonsCount() int {
	return s.seasonsCount
}

func (s *Show) EpisodesCount() int {
	return s.episodesCount
}

// Holds information about a season such as its index and episodes.
type Season struct {
	name     string
	index    int
	show     *Show
	episodes []*Episode
}

func (s *Season) Name() string {
	return s.name
}

func (s *Season) Index() int {
	return s.index
}

func (s *Season) Show() *Show {
	return s.show
}

func (s *Season) Episodes() []*Episode {
	return s.episodes
}

// Holds information about an episode such as its index and subtitles.
type Episode struct {
	file

	index  int
	season *Season
}

func (e *Episode) Index() int {
	return e.index
}

func (e *Episode) Name() string {
	return fmt.Sprintf("%s - S%02dE%02d",
		e.Season().Show().Name(),
		e.Season().Index(),
		e.Index(),
	)
}

func (e *Episode) FullName() string {
	return fmt.Sprintf("%s.%s",
		e.Name(),
		e.Extension(),
	)
}

func (e *Episode) Season() *Season {
	return e.season
}

// Finds season index in seasons array.
func findShowSeasonIndex(name string, seasons []*Season) int {
	seasonIndex := -1
	for i, season := range seasons {
		if season.Name() == name {
			seasonIndex = i
			continue
		}
	}

	return seasonIndex
}

// Finds Show index in Shows array.
func findShowIndex(name string, shows []*Show) int {
	showIndex := -1
	for i, show := range shows {
		if show.Name() == name {
			showIndex = i
			continue
		}
	}

	return showIndex
}
