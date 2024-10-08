package model

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/jeremiergz/nas-cli/internal/model/internal/mkv"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	ErrEmptyFilePath = errors.New("file path cannot be empty")
)

type MediaFile interface {
	Basename() string
	Clean() error
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
}

func newFile(basename, extension, filePath string) (*file, error) {
	if filePath == "" {
		return nil, ErrEmptyFilePath
	}
	return &file{
		basename:  basename,
		extension: extension,
		filePath:  filePath,
		id:        uuid.New(),
	}, nil
}

func (f *file) Basename() string {
	return f.basename
}

func (f *file) Clean() error {
	return mkv.CleanTracks(f.filePath, f.basename)
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
	f.filePath = path
}

func (f *file) Subtitles(languages ...string) map[string]string {
	subtitles := map[string]string{}

	files, err := os.ReadDir(filepath.Dir(f.FilePath()))
	if err == nil {
		videoFileNameLength := len(f.Name())
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
				isSubtitle := (videoFileNameLength + expectedSuffixSize) == len(filename)

				if isSubtitle {
					languageCode := filename[videoFileNameLength+1 : videoFileNameLength+4]
					if languages != nil && !slices.Contains(languages, languageCode) {
						continue
					}
					subtitles[languageCode] = filename
				}
			}
		}
	}

	return subtitles
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
func Files(wd string, extensions []string) ([]*File, error) {
	toProcess := fsutil.List(wd, extensions, nil)
	files := []*File{}
	for _, basename := range toProcess {
		extension := strings.TrimPrefix(filepath.Ext(basename), ".")

		f, err := newFile(basename, extension, filepath.Join(wd, basename))
		if err != nil {
			return nil, err
		}

		files = append(files, &File{
			file: f,
		})
	}
	return files, nil
}
