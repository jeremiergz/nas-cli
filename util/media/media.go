package media

import (
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strings"

	PTN "github.com/middelink/go-parse-torrent-name"
	"gitlab.com/jeremiergz/nas-cli/util"
)

// Episode holds information about an episode
type Episode struct {
	Basename  string
	Extension string
	Fullname  string
}

// Movie is the type of data that will be formatted as a movie
type Movie struct {
	Basename  string
	Extension string
	Fullname  string
	Title     string
	Year      int
}

// Season holds information about a season
type Season struct {
	Name     string
	Episodes []Episode
}

// TVShow is the type of data that will be formatted as a TV show
type TVShow struct {
	Name    string
	Seasons []Season
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
		shouldProcess := !f.IsDir() && isValidExt && !regExp.Match([]byte(f.Name()))
		if shouldProcess {
			filesList = append(filesList, f.Name())
		}
	}
	return filesList
}

// ParseTitle returns parsed information from a file name
func ParseTitle(filename string) (*PTN.TorrentInfo, error) {
	return PTN.Parse((filename))
}

// ToEpisodeName returns formatted TV show episode name from given parameters
func ToEpisodeName(title string, season int, episode int, container string) string {
	return fmt.Sprintf("%s - %dx%02d.%s", title, season, episode, container)
}

// ToMovieName returns formatted movie name from given parameters
func ToMovieName(title string, year int, extension string) string {
	return fmt.Sprintf("%s (%d).%s", title, year, extension)
}

// ToSeasonName returns formatted season name from given parameter
func ToSeasonName(season int) string {
	return fmt.Sprintf("Season %d", season)
}
