package service

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	ptn "github.com/middelink/go-parse-torrent-name"

	"github.com/jeremiergz/nas-cli/config"
	"github.com/jeremiergz/nas-cli/model"
	"github.com/jeremiergz/nas-cli/util"
)

type MediaService struct{}

var (
	tvShowFmtRegexp = regexp.MustCompile(`(^.+)(\s-\s)S\d+E\d+\.(.+)$`)
)

func NewMediaService() *MediaService {
	service := &MediaService{}

	return service
}

// Verifies that given path exists and sets WD variable
func (s *MediaService) InitializeWD(path string) error {
	config.WD, _ = filepath.Abs(path)
	stats, err := os.Stat(config.WD)
	if err != nil || !stats.IsDir() {
		return fmt.Errorf("%s is not a valid directory", config.WD)
	}

	return nil
}

// Lists files in directory with filter on extensions and RegExp
func (s *MediaService) List(wd string, extensions []string, regExp *regexp.Regexp) []string {
	files, _ := os.ReadDir(wd)
	filesList := []string{}
	for _, f := range files {
		ext := strings.Replace(path.Ext(f.Name()), ".", "", 1)
		isValidExt := util.StringInSlice(ext, extensions)
		shouldProcess := !f.IsDir() && isValidExt
		if shouldProcess {
			if regExp == nil || !regExp.Match([]byte(f.Name())) {
				filesList = append(filesList, f.Name())
			}
		}
	}

	return filesList
}

// Lists TV shows in folder that must be processed
func (s *MediaService) LoadTVShows(wd string, extensions []string, subtitlesExt *string, subtitlesLangs []string, anyFiles bool) ([]*model.TVShow, error) {
	var selectedRegexp *regexp.Regexp
	if !anyFiles {
		selectedRegexp = tvShowFmtRegexp
	}

	toProcess := s.List(wd, extensions, selectedRegexp)
	tvShows := []*model.TVShow{}
	for _, basename := range toProcess {
		e, err := s.ParseTitle(basename)
		e.Title = strings.Title(e.Title)
		if err == nil {
			var tvShow *model.TVShow
			tvShowIndex := s.findTVShowIndex(e.Title, tvShows)
			if tvShowIndex == -1 {
				tvShow = &model.TVShow{
					Name:    e.Title,
					Seasons: []*model.Season{},
				}
			} else {
				tvShow = tvShows[tvShowIndex]
			}
			seasonName := util.ToSeasonName(e.Season)
			seasonIndex := s.findSeasonIndex(seasonName, tvShow.Seasons)
			episode := model.Episode{
				Basename:  basename,
				Extension: e.Container,
				Index:     e.Episode,
			}
			episode.Subtitles = s.listSubtitles(basename, subtitlesExt, subtitlesLangs)
			var season *model.Season
			if seasonIndex == -1 {
				season = &model.Season{
					Episodes: []*model.Episode{},
					Index:    e.Season,
					Name:     seasonName,
					TVShow:   tvShow,
				}
				episode.Season = season
				season.Episodes = append(season.Episodes, &episode)
				tvShow.Seasons = append(tvShow.Seasons, season)
			} else {
				season := tvShow.Seasons[seasonIndex]
				episode.Season = season
				season.Episodes = append(season.Episodes, &episode)
			}
			if tvShowIndex == -1 {
				tvShows = append(tvShows, tvShow)
			}
		} else {
			return nil, err
		}
	}

	return tvShows, nil
}

// Returns parsed information from a file name
func (s *MediaService) ParseTitle(filename string) (*ptn.TorrentInfo, error) {
	return ptn.Parse((filename))
}

// Creates target directory, setting its mode to 755 and setting ownership
func (s *MediaService) PrepareDirectory(targetDirectory string, owner, group int) {
	os.Mkdir(targetDirectory, config.DirectoryMode)
	os.Chmod(targetDirectory, config.DirectoryMode)
	os.Chown(targetDirectory, owner, group)
}

// Finds season index in seasons array
func (s *MediaService) findSeasonIndex(name string, seasons []*model.Season) int {
	seasonIndex := -1
	for i, season := range seasons {
		if season.Name == name {
			seasonIndex = i
			continue
		}
	}

	return seasonIndex
}

// Finds TV Show index in TV Shows array
func (s *MediaService) findTVShowIndex(name string, tvShows []*model.TVShow) int {
	tvShowIndex := -1
	for i, tvShow := range tvShows {
		if tvShow.Name == name {
			tvShowIndex = i
			continue
		}
	}

	return tvShowIndex
}

// Lists subtitles with given extension & languages for the file passed as parameter
func (s *MediaService) listSubtitles(videoFile string, extension *string, languages []string) map[string]string {
	fileName := videoFile[:len(videoFile)-len(filepath.Ext(videoFile))]
	subtitles := map[string]string{}

	if extension != nil && languages != nil {
		for _, lang := range languages {
			subtitleFileName := fmt.Sprintf("%s.%s.%s", fileName, lang, *extension)
			subtitleFilePath, _ := filepath.Abs(path.Join(config.WD, subtitleFileName))
			stats, err := os.Stat(subtitleFilePath)
			if err == nil && !stats.IsDir() {
				subtitles[lang] = subtitleFileName
			}
		}
	}

	return subtitles
}
