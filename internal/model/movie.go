package model

import (
	"fmt"
	"path/filepath"
	"regexp"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/jeremiergz/nas-cli/internal/model/internal/parser"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	_ MediaFile = (*Movie)(nil)

	movieNameCaser = cases.Title(language.Und)
	movieFmtRegexp = regexp.MustCompile(`(^.+)\s\(([0-9]{4})\)\.(.+)$`)
)

// Holds information about a file parsed as a movie such as its title and year.
type Movie struct {
	file

	title string
	year  int
}

// Lists movies in given folder.
//
// Result can be filtered by extensions.
func Movies(wd string, extensions []string) ([]*Movie, error) {
	toProcess := fsutil.List(wd, extensions, movieFmtRegexp)
	movies := []*Movie{}
	for _, basename := range toProcess {
		parsed, err := parser.Parse(basename)
		if err == nil {
			movies = append(movies, &Movie{
				file: file{
					basename:  basename,
					extension: parsed.Container,
					filePath:  filepath.Join(wd, basename),
				},
				title: movieNameCaser.String(parsed.Title),
				year:  parsed.Year,
			})
		} else {
			return nil, err
		}
	}
	return movies, nil
}

func (m *Movie) Name() string {
	return m.title
}

func (m *Movie) SetName(name string) {
	m.title = name
}

func (m *Movie) FullName() string {
	return fmt.Sprintf("%s (%d)", m.title, m.year)
}

func (m *Movie) Year() int {
	return m.year
}

func (m *Movie) SetYear(year int) {
	m.year = year
}
