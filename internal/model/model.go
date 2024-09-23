package model

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/jeremiergz/nas-cli/internal/model/internal/mkv"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	ErrEmptyFilePath = errors.New("file path cannot be empty")
)

type MediaFile interface {
	Basename() string
	Clean() util.Result
	Extension() string
	FilePath() string
	SetFilePath(path string)
	Name() string
}

type file struct {
	basename  string
	extension string
	filePath  string
}

func (f *file) Basename() string {
	return f.basename
}

func (f *file) Clean() util.Result {
	return mkv.CleanTracks(f.filePath, f.basename)
}

func (f *file) Extension() string {
	return f.extension
}

func (f *file) FilePath() string {
	return f.filePath
}

func (f *file) SetFilePath(path string) {
	f.filePath = path
}

type File struct {
	file
}

// Lists files in given folder.
//
// Result can be filtered by extensions.
func Files(wd string, extensions []string) ([]*File, error) {
	toProcess := fsutil.List(wd, extensions, nil)
	files := []*File{}
	for _, basename := range toProcess {
		files = append(files, &File{
			file: file{
				basename:  basename,
				extension: basename,
				filePath:  filepath.Join(wd, basename),
			},
		})
	}
	return files, nil
}

// Lists subtitles with given extension and languages for the file passed as parameter.
func listSubtitles(wd, videoFile string, extension string, languages []string) map[string]string {
	fileName := videoFile[:len(videoFile)-len(filepath.Ext(videoFile))]
	subtitles := map[string]string{}

	if extension != "" && languages != nil {
		for _, lang := range languages {
			subtitleFilename := fmt.Sprintf("%s.%s.%s", fileName, lang, extension)
			subtitleFilePath, _ := filepath.Abs(path.Join(wd, subtitleFilename))
			stats, err := os.Stat(subtitleFilePath)
			if err == nil && !stats.IsDir() {
				subtitles[lang] = subtitleFilename
			}
		}
	}

	return subtitles
}
