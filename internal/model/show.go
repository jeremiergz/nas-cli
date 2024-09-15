package model

import (
	"github.com/jeremiergz/nas-cli/internal/util"
)

// Type of data that will be formatted as a show.
type Show struct {
	Name          string
	SeasonsCount  int
	EpisodesCount int
	Seasons       []*Season
}

// Season holds information about a season.
type Season struct {
	Episodes []*Episode
	Index    int
	Name     string
	Show     *Show
}

// Episode holds information about an episode.
type Episode struct {
	Basename  string
	Extension string
	FilePath  string
	Index     int
	Season    *Season
	Subtitles map[string]string
}

// Returns formatted show episode name from given parameters.
func (e *Episode) Name() string {
	return util.ToEpisodeName(e.Season.Show.Name, e.Season.Index, e.Index, e.Extension)
}
