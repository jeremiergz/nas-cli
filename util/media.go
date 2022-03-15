package util

import (
	"fmt"
)

// Returns formatted TV show episode name from given parameters
func ToEpisodeName(title string, season int, episode int, extension string) string {
	return fmt.Sprintf("%s - S%02dE%02d.%s", title, season, episode, extension)
}

// Returns formatted movie name from given parameters
func ToMovieName(title string, year int, extension string) string {
	return fmt.Sprintf("%s (%d).%s", title, year, extension)
}

// Returns formatted season name from given parameter
func ToSeasonName(season int) string {
	return fmt.Sprintf("Season %d", season)
}
