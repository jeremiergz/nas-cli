package model

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jeremiergz/nas-cli/internal/model/image"
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

	images []*image.Image
	title  string
	year   int
}

// Sorts movies by name in ascending order.
func SortMoviesByName(movies []*Movie) {
	slices.SortFunc(movies, func(a, b *Movie) int {
		return strings.Compare(a.Name(), b.Name())
	})
}

// Sorts movies by year in ascending order.
func SortMoviesByYear(movies []*Movie) {
	slices.SortFunc(movies, func(a, b *Movie) int {
		return a.Year() - b.Year()
	})
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
		m.Title = util.ToTitleCase(m.Title)

		if err == nil {
			filePath := filepath.Join(wd, path)
			f, err := newFile(basename, m.Container, filePath)
			if err != nil {
				return nil, err
			}

			referenceName := strings.TrimSuffix(basename, filepath.Ext(basename))
			baseImages, err := listBaseImageFiles(wd, referenceName)
			if err != nil {
				return nil, fmt.Errorf("failed to list movie images for %s: %w", referenceName, err)
			}

			movies = append(movies, &Movie{
				file:   f,
				images: baseImages,
				title:  m.Title,
				year:   m.Year,
			})
		} else {
			return nil, err
		}
	}
	return movies, nil
}

func (m *Movie) Images() []*image.Image {
	return m.images
}

func (m *Movie) ConvertImagesToRequirements() error {
	for i, img := range m.images {
		newPath, err := convertImageFileToRequirements(img.FilePath, img.Kind)
		if err != nil {
			return fmt.Errorf("failed to convert movie %s image file %s: %w", img.Kind, img.FilePath, err)
		}
		m.images[i].FilePath = newPath
	}
	return nil
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
