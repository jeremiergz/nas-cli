package model

import (
	"fmt"
	"path/filepath"

	"github.com/jeremiergz/nas-cli/internal/model/internal/parser"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	_ MediaFile = (*Movie)(nil)
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
		m, err := parser.Parse(basename)
		m.Title = util.ToUpperFirst(m.Title)

		if err == nil {
			f, err := newFile(basename, m.Container, filepath.Join(wd, path))
			if err != nil {
				return nil, err
			}

			movies = append(movies, &Movie{
				file:  f,
				title: m.Title,
				year:  m.Year,
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
	return fmt.Sprintf(
		"%s (%d).%s",
		m.Name(),
		m.Year(),
		m.Extension(),
	)
}

func (m *Movie) Year() int {
	return m.year
}

func (m *Movie) SetYear(year int) {
	m.year = year
}
