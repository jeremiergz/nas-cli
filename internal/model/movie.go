package model

import (
	"fmt"
	"path/filepath"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/jeremiergz/nas-cli/internal/model/internal/parser"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	_ MediaFile = (*Movie)(nil)

	movieNameCaser = cases.Title(language.Und)
)

// Holds information about a file parsed as a movie such as its title and year.
type Movie struct {
	*file

	title string
	year  int
}

// Lists movies in given folder.
//
// Result can be filtered by extensions.
func Movies(wd string, extensions []string, recursive bool) ([]*Movie, error) {
	toProcess := fsutil.List(wd, extensions, nil, recursive)
	movies := []*Movie{}
	for _, path := range toProcess {
		basename := filepath.Base(path)
		parsed, err := parser.Parse(basename)
		if err == nil {
			f, err := newFile(basename, parsed.Container, filepath.Join(wd, path))
			if err != nil {
				return nil, err
			}

			movies = append(movies, &Movie{
				file:  f,
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
