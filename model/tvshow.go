package model

// TVShow is the type of data that will be formatted as a TV show
type TVShow struct {
	Name    string
	Seasons []*Season
}
