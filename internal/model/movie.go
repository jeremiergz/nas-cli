package model

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/image/webp"

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

	title               string
	year                int
	backgroundImagePath *string
	posterImagePath     *string
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

			backgroundImg, posterImg, err := listMovieImageFiles(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to list movie images for %s: %w", filePath, err)
			}

			movies = append(movies, &Movie{
				file:                f,
				title:               m.Title,
				year:                m.Year,
				backgroundImagePath: backgroundImg,
				posterImagePath:     posterImg,
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

func (m *Movie) BackgroundImageFilePath() *string {
	return m.backgroundImagePath
}

func (m *Movie) PosterImageFilePath() *string {
	return m.posterImagePath
}

func listMovieImageFiles(movieFilePath string) (background, poster *string, err error) {
	files, err := os.ReadDir(filepath.Dir(movieFilePath))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read directory: %w", err)
	}

	validFileExtensions := []string{"jpg", "jpeg", "png", "webp"}

	movieName := strings.TrimSuffix(filepath.Base(movieFilePath), filepath.Ext(movieFilePath))

	for _, file := range files {
		filePath := filepath.Join(".", file.Name())
		fileName := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		fileExtension := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Name()), "."))

		hasValidExtension := slices.Contains(validFileExtensions, fileExtension)
		hasMovieName := strings.HasPrefix(fileName, movieName)

		if hasValidExtension && hasMovieName {
			if slices.Contains([]string{"background", "bg"}, fileName) {
				background = &filePath
				err = processMovieImageFile(filePath)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to process background image file %s: %w", filePath, err)
				}
			}

			if slices.Contains([]string{"poster", "pt"}, fileName) {
				poster = &filePath
				err = processMovieImageFile(filePath)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to process poster image file %s: %w", filePath, err)
				}
			}
		}

		if background != nil && poster != nil {
			break
		}
	}

	return background, poster, nil
}

func processMovieImageFile(src string) error {
	imgName := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	imgExtension := strings.ToLower(strings.TrimPrefix(filepath.Ext(src), "."))

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open image file: %w", err)
	}
	defer srcFile.Close()

	var decoded image.Image

	switch imgExtension {
	case "jpg", "jpeg":
		if imgExtension == "jpeg" {
			err = os.Rename(src, imgName+".jpg")
			if err != nil {
				return fmt.Errorf("failed to rename jpeg image file: %w", err)
			}
		}
		return nil

	case "png":
		decoded, err = png.Decode(srcFile)
		if err != nil {
			return fmt.Errorf("failed to decode png image file: %w", err)
		}

	case "webp":
		decoded, err = webp.Decode(srcFile)
		if err != nil {
			return fmt.Errorf("failed to decode webp image file: %w", err)
		}

	default:
		return fmt.Errorf("unsupported image file format: %s", imgExtension)
	}

	outputFile, err := os.Create(imgName + ".jpg")
	if err != nil {
		return fmt.Errorf("failed to create output image file: %w", err)
	}
	defer outputFile.Close()

	err = jpeg.Encode(outputFile, decoded, &jpeg.Options{Quality: 90})
	if err != nil {
		return fmt.Errorf("failed to encode jpeg image file: %w", err)
	}

	err = os.Remove(src)
	if err != nil {
		return fmt.Errorf("failed to remove original image file: %w", err)
	}

	return nil
}
