package model

// Season holds information about a season
type Season struct {
	Episodes []*Episode
	Index    int
	Name     string
	TVShow   *TVShow
}
