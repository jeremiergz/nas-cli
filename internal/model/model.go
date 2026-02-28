package model

import (
	"context"
	"errors"
	"fmt"
	goimage "image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/image/webp"

	"github.com/jeremiergz/nas-cli/internal/model/image"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	ErrEmptyFilePath = errors.New("file path cannot be empty")
)

type Kind string

const (
	KindAnime  Kind = "anime"
	KindMovie  Kind = "movie"
	KindTVShow Kind = "tvshow"
)

func (k Kind) String() string {
	return string(k)
}

type MediaFile interface {
	Basename() string
	Extension() string
	FilePath() string
	FullName() string
	ID() uuid.UUID
	Name() string
	SetFilePath(path string)
	Subtitles(languages ...string) map[string]string
}

type file struct {
	basename  string
	extension string
	filePath  string
	id        uuid.UUID
	subtitles map[string]string
}

func newFile(basename, extension, filePath string) (*file, error) {
	if filePath == "" {
		return nil, ErrEmptyFilePath
	}
	return &file{
		id:        uuid.New(),
		basename:  basename,
		extension: extension,
		filePath:  filePath,
	}, nil
}

func (f *file) Basename() string {
	return f.basename
}

func (f *file) Extension() string {
	return f.extension
}

func (f *file) FilePath() string {
	return f.filePath
}

func (f *file) FullName() string {
	return f.basename
}

func (f *file) ID() uuid.UUID {
	return f.id
}

func (f *file) Name() string {
	return f.basename[:len(f.basename)-len(filepath.Ext(f.basename))]
}

func (f *file) SetFilePath(path string) {
	if path == "" {
		panic(ErrEmptyFilePath)
	}
	f.basename = filepath.Base(path)
	f.extension = strings.TrimPrefix(filepath.Ext(path), ".")
	f.filePath = path
}

func (f *file) Subtitles(languages ...string) map[string]string {
	if f.subtitles == nil {
		subtitles := map[string]string{}

		files, err := os.ReadDir(filepath.Dir(f.FilePath()))
		if err == nil {
			videoFilenameLength := len(f.Name())
			subtitleExtension := fmt.Sprintf(".%s", util.AcceptedSubtitleExtension)

			// We look for files with the same name as the video file, the .srt extension
			// and a 3-letter language code. E.g.: video.eng.srt, video.spa.srt.
			expectedSuffixSize := 4 + len(subtitleExtension)

			for _, file := range files {
				if file.IsDir() {
					continue
				}

				filename := file.Name()
				isValidExtension := filepath.Ext(filename) == subtitleExtension

				if isValidExtension {
					isSubtitle := (videoFilenameLength + expectedSuffixSize) == len(filename)

					if isSubtitle {
						languageCode := filename[videoFilenameLength+1 : videoFilenameLength+4]
						subtitleName := filename[:len(filename)-expectedSuffixSize]
						if languages != nil && !slices.Contains(languages, languageCode) {
							continue
						}
						if subtitleName == f.Name() {
							subtitles[languageCode] = filename
						}
					}
				}
			}
		}
		f.subtitles = subtitles
	}

	return f.subtitles
}

var (
	_ MediaFile = (*File)(nil)
)

type File struct {
	*file
}

// Lists files in given folder.
//
// Result can be filtered by extensions.
func Files(wd string, extensions []string, recursive bool) ([]*File, error) {
	toProcess := fsutil.List(wd, extensions, nil, recursive)
	files := []*File{}
	for _, path := range toProcess {
		basename := filepath.Base(path)
		extension := strings.TrimPrefix(filepath.Ext(basename), ".")

		f, err := newFile(basename, extension, filepath.Join(wd, path))
		if err != nil {
			return nil, err
		}

		files = append(files, &File{
			file: f,
		})
	}
	return files, nil
}

func listBaseImageFiles(dir, referenceName string) (imageFiles []*image.Image, err error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Keep only image files to reduce the number of iterations later on.
	files = slices.DeleteFunc(files, func(file os.DirEntry) bool {
		fileExtension := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Name()), "."))
		return !slices.Contains(image.ValidExtensions, fileExtension)
	})

	for _, file := range files {
		filePath := filepath.Join(".", file.Name())
		fileName := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		hasReferenceName := strings.HasPrefix(fileName, referenceName)

		if hasReferenceName {
			isBackgroundImage := strings.HasSuffix(fileName, ".background") || strings.HasSuffix(fileName, ".bg")
			if isBackgroundImage {
				imageFiles = append(imageFiles, image.New("background", filePath, image.KindBackground))
			}
			isPosterImage := strings.HasSuffix(fileName, ".poster") || strings.HasSuffix(fileName, ".pt")
			if isPosterImage {
				imageFiles = append(imageFiles, image.New("poster", filePath, image.KindPoster))
			}
		}

		if len(imageFiles) == 2 {
			break
		}
	}

	return imageFiles, nil
}

// Converts the image file to meet the requirements of the media server (dimensions, format, DPI) depending on the kind
// of image (background or poster).
func convertImageFileToRequirements(src string, kind image.Kind) (string, error) {
	imgName := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	imgExtension := strings.ToLower(strings.TrimPrefix(filepath.Ext(src), "."))

	srcFile, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("failed to open image file: %w", err)
	}
	defer srcFile.Close()

	var decoded goimage.Image
	shouldEncode := false
	shouldDeleteSourceFile := false

	switch imgExtension {
	case "jpg", "jpeg":
		decoded, err = jpeg.Decode(srcFile)
		if err != nil {
			return "", fmt.Errorf("failed to decode jpeg image file: %w", err)
		}
		if imgExtension == "jpeg" {
			err = os.Rename(src, imgName+".jpg")
			if err != nil {
				return "", fmt.Errorf("failed to rename jpeg image file: %w", err)
			}
		}

	case "png":
		decoded, err = png.Decode(srcFile)
		if err != nil {
			return "", fmt.Errorf("failed to decode png image file: %w", err)
		}
		shouldEncode = true
		shouldDeleteSourceFile = true

	case "webp":
		decoded, err = webp.Decode(srcFile)
		if err != nil {
			return "", fmt.Errorf("failed to decode webp image file: %w", err)
		}
		shouldEncode = true
		shouldDeleteSourceFile = true

	default:
		return "", fmt.Errorf("unsupported image file format: %s", imgExtension)
	}

	// Check if the image has the desired dimensions.
	var expectedX, expectedY int
	switch kind {
	case image.KindBackground:
		expectedX = image.KindBackgroundWidth
		expectedY = image.KindBackgroundHeight
	case image.KindPoster:
		expectedX = image.KindPosterWidth
		expectedY = image.KindPosterHeight
	}
	currentX := decoded.Bounds().Dx()
	currentY := decoded.Bounds().Dy()
	if currentX != expectedX || currentY != expectedY {
		shouldEncode = true
	}

	outputFilePath := imgName + ".jpg"

	if shouldEncode {
		decoded = image.Scale(decoded, kind)

		outputFile, err := os.Create(outputFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to create output image file: %w", err)
		}
		defer outputFile.Close()

		err = jpeg.Encode(outputFile, decoded, &jpeg.Options{Quality: 90})
		if err != nil {
			return "", fmt.Errorf("failed to encode jpeg image file: %w", err)
		}
	}

	if shouldDeleteSourceFile {
		err = os.Remove(src)
		if err != nil {
			return "", fmt.Errorf("failed to remove original image file: %w", err)
		}
	}

	err = image.SetDPI(context.Background(), outputFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to set DPI for image file: %w", err)
	}

	return outputFilePath, nil
}
