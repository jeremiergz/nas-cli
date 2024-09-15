package media

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	"github.com/jeremiergz/nas-cli/internal/service/media/internal"
	"github.com/jeremiergz/nas-cli/internal/util"
)

type Service struct{}

var (
	episodeNameCaser = cases.Title(language.Und)
	movieFmtRegexp   = regexp.MustCompile(`(^.+)\s\(([0-9]{4})\)\.(.+)$`)
	showFmtRegexp    = regexp.MustCompile(`(^.+)(\s-\s)S\d+E\d+\.(.+)$`)
)

func New() *Service {
	return &Service{}
}

// Verifies that given path exists and sets WD variable.
func (s *Service) InitializeWD(path string) error {
	config.WD, _ = filepath.Abs(path)
	stats, err := os.Stat(config.WD)
	if err != nil || !stats.IsDir() {
		return fmt.Errorf("%s is not a valid directory", config.WD)
	}

	return nil
}

// Lists files in directory with filter on extensions and RegExp.
func (s *Service) List(wd string, extensions []string, regExp *regexp.Regexp) []string {
	files, _ := os.ReadDir(wd)
	filesList := []string{}
	for _, f := range files {
		ext := strings.Replace(path.Ext(f.Name()), ".", "", 1)
		isValidExt := slices.Contains(extensions, ext)
		shouldProcess := !f.IsDir() && isValidExt
		if shouldProcess {
			if regExp == nil || !regExp.Match([]byte(f.Name())) {
				filesList = append(filesList, f.Name())
			}
		}
	}

	return filesList
}

// Lists movies in folder that must be processed
func (s *Service) LoadMovies(wd string, extensions []string) ([]*model.Movie, error) {
	toProcess := s.List(wd, extensions, movieFmtRegexp)
	movies := []*model.Movie{}
	for _, basename := range toProcess {
		m, err := s.ParseTitle(basename)
		if err == nil {
			movies = append(movies, &model.Movie{
				Basename:  basename,
				Extension: m.Container,
				Fullname:  util.ToMovieName(m.Title, m.Year, m.Container),
				Title:     m.Title,
				Year:      m.Year,
			})
		} else {
			return nil, err
		}
	}
	return movies, nil
}

// Lists shows in folder that must be processed.
func (s *Service) LoadShows(wd string, extensions []string, subtitlesExt *string, subtitlesLangs []string, anyFiles bool) ([]*model.Show, error) {
	var selectedRegexp *regexp.Regexp
	if !anyFiles {
		selectedRegexp = showFmtRegexp
	}

	toProcess := s.List(wd, extensions, selectedRegexp)
	shows := []*model.Show{}
	for _, basename := range toProcess {
		e, err := s.ParseTitle(basename)

		e.Title = episodeNameCaser.String(e.Title)
		if err == nil {
			var show *model.Show
			showIndex := s.findShowIndex(e.Title, shows)
			if showIndex == -1 {
				show = &model.Show{
					Name:    e.Title,
					Seasons: []*model.Season{},
				}
			} else {
				show = shows[showIndex]
			}
			seasonName := util.ToSeasonName(e.Season)
			seasonIndex := s.findSeasonIndex(seasonName, show.Seasons)
			episode := model.Episode{
				Basename:  basename,
				FilePath:  path.Join(wd, basename),
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
					Show:     show,
				}
				episode.Season = season
				season.Episodes = append(season.Episodes, &episode)
				show.Seasons = append(show.Seasons, season)
			} else {
				season := show.Seasons[seasonIndex]
				episode.Season = season
				season.Episodes = append(season.Episodes, &episode)
			}
			if showIndex == -1 {
				shows = append(shows, show)
			}
		} else {
			return nil, err
		}
	}

	for _, show := range shows {
		show.SeasonsCount = len(show.Seasons)
		for _, season := range show.Seasons {
			show.EpisodesCount += len(season.Episodes)
		}
	}

	return shows, nil
}

// Returns parsed information from a file name.
func (s *Service) ParseTitle(filename string) (*internal.DownloadedFile, error) {
	return internal.Parse((filename))
}

// Creates target directory, setting its mode to 755 and setting ownership.
func (s *Service) PrepareDirectory(targetDirectory string, owner, group int) {
	os.Mkdir(targetDirectory, config.DirectoryMode)
	os.Chmod(targetDirectory, config.DirectoryMode)
	os.Chown(targetDirectory, owner, group)
}

// Finds season index in seasons array.
func (s *Service) findSeasonIndex(name string, seasons []*model.Season) int {
	seasonIndex := -1
	for i, season := range seasons {
		if season.Name == name {
			seasonIndex = i
			continue
		}
	}

	return seasonIndex
}

// Finds Show index in Shows array.
func (s *Service) findShowIndex(name string, shows []*model.Show) int {
	showIndex := -1
	for i, show := range shows {
		if show.Name == name {
			showIndex = i
			continue
		}
	}

	return showIndex
}

// Lists subtitles with given extension & languages for the file passed as parameter.
func (s *Service) listSubtitles(videoFile string, extension *string, languages []string) map[string]string {
	fileName := videoFile[:len(videoFile)-len(filepath.Ext(videoFile))]
	subtitles := map[string]string{}

	if extension != nil && languages != nil {
		for _, lang := range languages {
			subtitleFilename := fmt.Sprintf("%s.%s.%s", fileName, lang, *extension)
			subtitleFilePath, _ := filepath.Abs(path.Join(config.WD, subtitleFilename))
			stats, err := os.Stat(subtitleFilePath)
			if err == nil && !stats.IsDir() {
				subtitles[lang] = subtitleFilename
			}
		}
	}

	return subtitles
}
