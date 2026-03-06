package media

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/pterm/pterm"
	"github.com/samber/lo"

	"github.com/jeremiergz/nas-cli/internal/image"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
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

// Prints given files array as a tree.
func PrintFiles(wd string, files []*File) {
	lw := cmdutil.NewListWriter()
	filesCount := len(files)

	lw.AppendItem(
		fmt.Sprintf(
			"%s (%d %s)",
			wd,
			filesCount,
			lo.Ternary(filesCount <= 1, "file", "files"),
		),
	)

	lw.Indent()
	for _, f := range files {
		lw.AppendItem(f.Basename())
	}

	pterm.Println(lw.Render())
}
