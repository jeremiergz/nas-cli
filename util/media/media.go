package media

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	ptn "github.com/middelink/go-parse-torrent-name"

	"github.com/jeremiergz/nas-cli/util"
)

// TVShow is the type of data that will be formatted as a TV show
type TVShow struct {
	Name    string
	Seasons []*Season
}

// Season holds information about a season
type Season struct {
	Episodes []*Episode
	Index    int
	Name     string
	TVShow   *TVShow
}

// Episode holds information about an episode
type Episode struct {
	Basename  string
	Extension string
	Index     int
	Season    *Season
	Subtitles map[string]string
}

// Returns formatted TV show episode name from given parameters
func (e *Episode) Name() string {
	return ToEpisodeName(e.Season.TVShow.Name, e.Season.Index, e.Index, e.Extension)
}

// Movie is the type of data that will be formatted as a movie
type Movie struct {
	Basename  string
	Extension string
	Fullname  string
	Title     string
	Year      int
}

var (
	// GID is the processed files group to set
	GID int

	// UID is the processed files owner to set
	UID int

	// WD is the working directory's absolute path
	WD string
)

var tvShowFmtRegexp = regexp.MustCompile(`(^.+)(\s-\s)S\d+E\d+\.(.+)$`)

// Verifies that given path exists and sets WD variable
func InitializeWD(path string) error {
	WD, _ = filepath.Abs(path)
	stats, err := os.Stat(WD)
	if err != nil || !stats.IsDir() {
		return fmt.Errorf("%s is not a valid directory", WD)
	}

	return nil
}

// Lists files in directory with filter on extensions and RegExp
func List(wd string, extensions []string, regExp *regexp.Regexp) []string {
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
func LoadTVShows(wd string, extensions []string, subtitlesExt *string, subtitlesLangs []string, anyFiles bool) ([]*TVShow, error) {
	var selectedRegexp *regexp.Regexp
	if !anyFiles {
		selectedRegexp = tvShowFmtRegexp
	}

	toProcess := List(wd, extensions, selectedRegexp)
	tvShows := []*TVShow{}
	for _, basename := range toProcess {
		e, err := ParseTitle(basename)
		e.Title = strings.Title(e.Title)
		if err == nil {
			var tvShow *TVShow
			tvShowIndex := findTVShowIndex(e.Title, tvShows)
			if tvShowIndex == -1 {
				tvShow = &TVShow{
					Name:    e.Title,
					Seasons: []*Season{},
				}
			} else {
				tvShow = tvShows[tvShowIndex]
			}
			seasonName := ToSeasonName(e.Season)
			seasonIndex := findSeasonIndex(seasonName, tvShow.Seasons)
			episode := Episode{
				Basename:  basename,
				Extension: e.Container,
				Index:     e.Episode,
			}
			episode.Subtitles = listSubtitles(basename, subtitlesExt, subtitlesLangs)
			var season *Season
			if seasonIndex == -1 {
				season = &Season{
					Episodes: []*Episode{},
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
func ParseTitle(filename string) (*ptn.TorrentInfo, error) {
	return ptn.Parse((filename))
}

// Creates target directory, setting its mode to 755 and setting ownership
func PrepareDirectory(targetDirectory string, owner, group int) {
	os.Mkdir(targetDirectory, util.DirectoryMode)
	os.Chmod(targetDirectory, util.DirectoryMode)
	os.Chown(targetDirectory, owner, group)
}

// Returns formatted TV show episode name from given parameters
func ToEpisodeName(title string, season int, episode int, extension string) string {
	return fmt.Sprintf("%s - S%02dE%02d.%s", title, season, episode, extension)
}

// Returns formatted movie name from given parameters
func ToMovieName(title string, year int, extension string) string {
	return fmt.Sprintf("%s (%d).%s", title, year, extension)
}

// Returns formatted season name from given parameter
func ToSeasonName(season int) string {
	return fmt.Sprintf("Season %d", season)
}

// Finds season index in seasons array
func findSeasonIndex(name string, seasons []*Season) int {
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
func findTVShowIndex(name string, tvShows []*TVShow) int {
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
func listSubtitles(videoFile string, extension *string, languages []string) map[string]string {
	fileName := videoFile[:len(videoFile)-len(filepath.Ext(videoFile))]
	subtitles := map[string]string{}

	if extension != nil && languages != nil {
		for _, lang := range languages {
			subtitleFileName := fmt.Sprintf("%s.%s.%s", fileName, lang, *extension)
			subtitleFilePath, _ := filepath.Abs(path.Join(WD, subtitleFileName))
			stats, err := os.Stat(subtitleFilePath)
			if err == nil && !stats.IsDir() {
				subtitles[lang] = subtitleFileName
			}
		}
	}

	return subtitles
}
