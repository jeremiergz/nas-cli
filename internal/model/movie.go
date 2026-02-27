package model

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pterm/pterm"
	"golang.org/x/image/webp"

	"github.com/jeremiergz/nas-cli/internal/model/internal/parser"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
	"github.com/jeremiergz/nas-cli/internal/util/imgutil"
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

type imageFileToProcess struct {
	path string
	kind imgutil.ImageKind
}

var validImageFileExtensions = []string{"jpg", "jpeg", "png", "webp"}

func listMovieImageFiles(movieFilePath string) (background, poster *string, err error) {
	files, err := os.ReadDir(filepath.Dir(movieFilePath))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read directory: %w", err)
	}

	movieName := strings.TrimSuffix(filepath.Base(movieFilePath), filepath.Ext(movieFilePath))
	var imageFilesToProcess []imageFileToProcess

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
				imageFilesToProcess = append(imageFilesToProcess, imageFileToProcess{path: filePath, kind: imgutil.ImageKindBackground})
			}

			isPosterImage := strings.HasSuffix(fileName, ".poster") || strings.HasSuffix(fileName, ".pt")
			if isPosterImage {
				poster = &filePath
				imageFilesToProcess = append(imageFilesToProcess, imageFileToProcess{path: filePath, kind: imgutil.ImageKindPoster})
			}
		}

		if background != nil && poster != nil {
			break
		}
	}

	if len(imageFilesToProcess) > 0 {
		spinner, err := pterm.DefaultSpinner.Start("Processing images...")
		if err != nil {
			return nil, nil, fmt.Errorf("could not start spinner: %w", err)
		}
		for _, img := range imageFilesToProcess {
			err = processMovieImageFile(img.path, img.kind)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to process %s image file %s: %w", img.kind, img.path, err)
			}
		}
		if err := spinner.Stop(); err != nil {
			return nil, nil, fmt.Errorf("could not stop spinner: %w", err)
		}
	}

	return background, poster, nil
}

func processMovieImageFile(src string, kind imgutil.ImageKind) error {
	imgName := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	imgExtension := strings.ToLower(strings.TrimPrefix(filepath.Ext(src), "."))

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open image file: %w", err)
	}
	defer srcFile.Close()

	var decoded image.Image
	shouldEncode := false
	shouldDeleteSourceFile := false

	switch imgExtension {
	case "jpg", "jpeg":
		decoded, err = jpeg.Decode(srcFile)
		if err != nil {
			return fmt.Errorf("failed to decode jpeg image file: %w", err)
		}
		if imgExtension == "jpeg" {
			err = os.Rename(src, imgName+".jpg")
			if err != nil {
				return fmt.Errorf("failed to rename jpeg image file: %w", err)
			}
		}

	case "png":
		decoded, err = png.Decode(srcFile)
		if err != nil {
			return fmt.Errorf("failed to decode png image file: %w", err)
		}
		shouldEncode = true
		shouldDeleteSourceFile = true

	case "webp":
		decoded, err = webp.Decode(srcFile)
		if err != nil {
			return fmt.Errorf("failed to decode webp image file: %w", err)
		}
		shouldEncode = true
		shouldDeleteSourceFile = true

	default:
		return fmt.Errorf("unsupported image file format: %s", imgExtension)
	}

	// Check if the image has the desired dimensions.
	var expectedX, expectedY int
	switch kind {
	case imgutil.ImageKindBackground:
		expectedX = imgutil.ImageKindBackgroundWidth
		expectedY = imgutil.ImageKindBackgroundHeight
	case imgutil.ImageKindPoster:
		expectedX = imgutil.ImageKindPosterWidth
		expectedY = imgutil.ImageKindPosterHeight
	}
	currentX := decoded.Bounds().Dx()
	currentY := decoded.Bounds().Dy()
	if currentX != expectedX || currentY != expectedY {
		shouldEncode = true
	}

	outputFilePath := imgName + ".jpg"

	if shouldEncode {
		decoded = imgutil.Scale(decoded, kind)

		outputFile, err := os.Create(outputFilePath)
		if err != nil {
			return fmt.Errorf("failed to create output image file: %w", err)
		}
		defer outputFile.Close()

		err = jpeg.Encode(outputFile, decoded, &jpeg.Options{Quality: 90})
		if err != nil {
			return fmt.Errorf("failed to encode jpeg image file: %w", err)
		}
	}

	if shouldDeleteSourceFile {
		err = os.Remove(src)
		if err != nil {
			return fmt.Errorf("failed to remove original image file: %w", err)
		}
	}

	err = imgutil.SetDPI(context.Background(), outputFilePath)
	if err != nil {
		return fmt.Errorf("failed to set DPI for image file: %w", err)
	}

	return nil
}
