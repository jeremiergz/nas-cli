package model

import (
	"fmt"
	"os"
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

	title string
	year  int
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

			backgroundImgPath, posterImgPath, err := listMovieImageFiles(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to list movie images for %s: %w", filePath, err)
			}

			if backgroundImgPath != nil {
				f.images = append(f.images, image.New("background", *backgroundImgPath, image.KindBackground))
			}
			if posterImgPath != nil {
				f.images = append(f.images, image.New("poster", *posterImgPath, image.KindPoster))
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

func (m *Movie) Images() []*image.Image {
	return m.images
}

var validImageFileExtensions = []string{"jpg", "jpeg", "png", "webp"}

func listMovieImageFiles(movieFilePath string) (background, poster *string, err error) {
	files, err := os.ReadDir(filepath.Dir(movieFilePath))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read directory: %w", err)
	}

	movieName := strings.TrimSuffix(filepath.Base(movieFilePath), filepath.Ext(movieFilePath))

	for _, file := range files {
		filePath := filepath.Join(".", file.Name())
		fileName := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		fileExtension := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Name()), "."))

		hasValidExtension := slices.Contains(validImageFileExtensions, fileExtension)
		hasMovieName := strings.HasPrefix(fileName, movieName)

		if hasValidExtension && hasMovieName {
			isBackgroundImage := strings.HasSuffix(fileName, ".background") || strings.HasSuffix(fileName, ".bg")
			if isBackgroundImage {
				background = &filePath
			}

			isPosterImage := strings.HasSuffix(fileName, ".poster") || strings.HasSuffix(fileName, ".pt")
			if isPosterImage {
				poster = &filePath
			}
		}

		if background != nil && poster != nil {
			break
		}
	}

	return background, poster, nil
}
