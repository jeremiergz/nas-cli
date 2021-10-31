package media

import (
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strings"

	"github.com/jeremiergz/nas-cli/util"
	PTN "github.com/middelink/go-parse-torrent-name"
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
}

// Name returns formatted TV show episode name from given parameters
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

// Result is struct defining result of a process
type Result struct {
	IsSuccessful bool
	Message      string
}

var (
	// GID is the processed files group to set
	GID int

	// UID is the processed files owner to set
	UID int

	// WD is the working directory's absolute path
	WD string
)

// List lists files in directory with filter on extensions and RegExp
func List(wd string, extensions []string, regExp *regexp.Regexp) []string {
	files, _ := ioutil.ReadDir(wd)
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

// ParseTitle returns parsed information from a file name
func ParseTitle(filename string) (*PTN.TorrentInfo, error) {
	return PTN.Parse((filename))
}

// ToEpisodeName returns formatted TV show episode name from given parameters
func ToEpisodeName(title string, season int, episode int, extension string) string {
	return fmt.Sprintf("%s - S%02dE%02d.%s", strings.Title(title), season, episode, extension)
}

// ToMovieName returns formatted movie name from given parameters
func ToMovieName(title string, year int, extension string) string {
	return fmt.Sprintf("%s (%d).%s", title, year, extension)
}

// ToSeasonName returns formatted season name from given parameter
func ToSeasonName(season int) string {
	return fmt.Sprintf("Season %d", season)
}
