package model

import (
	"github.com/jeremiergz/nas-cli/util"
)

// Episode holds information about an episode
type Episode struct {
	Basename  string
	Extension string
	Index     int
	Season    *Season
	Subtitles map[string]string
}

// Returns formatted TV show episode name from given parameters
func (e *Episode) Name() string {
	return util.ToEpisodeName(e.Season.TVShow.Name, e.Season.Index, e.Index, e.Extension)
}
