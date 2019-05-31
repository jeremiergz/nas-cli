package tvshow

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"

	gotree "github.com/DiSiqueira/GoTree"
	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"gitlab.com/jeremiergz/nas-cli/util"
	"gitlab.com/jeremiergz/nas-cli/util/media"
	"gitlab.com/jeremiergz/nas-cli/util/openfaas"
)

// Episode holds information about an episode
type Episode struct {
	Basename  string
	Extension string
	Fullname  string
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

var tvShowFmtRegexp = regexp.MustCompile(`(^.+)(\s-\s)\d+x\d+\.(.+)$`)

func findSeasonIndex(name string, seasons []Season) int {
	seasonIndex := -1
	for i, season := range seasons {
		if season.Name == name {
			seasonIndex = i
			continue
		}
	}
	return seasonIndex
}

func findTVShowIndex(name string, tvShows []TVShow) int {
	tvShowIndex := -1
	for i, tvShow := range tvShows {
		if tvShow.Name == name {
			tvShowIndex = i
			continue
		}
	}
	return tvShowIndex
}

// loadTVShows lists TV shows in folder that must be processed
func loadTVShows(wd string, extensions []string) ([]TVShow, error) {
	toProcess := media.List(wd, extensions, tvShowFmtRegexp)

	jsonBody, err := json.Marshal(toProcess)
	responseData, err := openfaas.InvokeFaaS(openfaas.ParseMediaTitle, jsonBody)
	if err != nil {
		return nil, err
	}

	var parsedJSONResponse []struct {
		Basename  string
		Container string
		Episode   int
		Season    int
		Title     string
	}
	err = json.Unmarshal(responseData, &parsedJSONResponse)
	if err != nil {
		return nil, err
	}

	tvShows := []TVShow{}
	for _, e := range parsedJSONResponse {
		episode := Episode{
			Basename:  e.Basename,
			Extension: e.Container,
			Fullname:  fmt.Sprintf("%s - %dx%02d.%s", e.Title, e.Season, e.Episode, e.Container),
		}
		var tvShow *TVShow
		tvShowIndex := findTVShowIndex(e.Title, tvShows)
		if tvShowIndex == -1 {
			tvShow = &TVShow{
				Name:    e.Title,
				Seasons: []Season{},
			}
		} else {
			tvShow = &tvShows[tvShowIndex]
		}

		seasonName := fmt.Sprintf("Season %d", e.Season)
		seasonIndex := findSeasonIndex(seasonName, tvShow.Seasons)
		if seasonIndex == -1 {
			season := Season{
				Name:     seasonName,
				Episodes: []Episode{},
			}
			season.Episodes = append(season.Episodes, episode)
			tvShow.Seasons = append(tvShow.Seasons, season)
		} else {
			season := &tvShow.Seasons[seasonIndex]
			season.Episodes = append(season.Episodes, episode)
		}

		if tvShowIndex == -1 {
			tvShows = append(tvShows, *tvShow)
		}
	}
	return tvShows, nil
}

// printAll prints given TV shows as a tree
func printAll(wd string, tvShows []TVShow) {
	rootTree := gotree.New(wd)
	for _, tvShow := range tvShows {
		tvShowTree := rootTree.Add(tvShow.Name)
		for _, season := range tvShow.Seasons {
			seasonsTree := tvShowTree.Add(season.Name)
			for _, episode := range season.Episodes {
				seasonsTree.Add(fmt.Sprintf("%s  %s", episode.Fullname, aurora.Gray(10, episode.Basename)))
			}
		}
	}
	toPrint := rootTree.Print()
	lastSpaceRegexp := regexp.MustCompile(`\s$`)
	toPrint = lastSpaceRegexp.ReplaceAllString(toPrint, "")
	fmt.Println(toPrint)
}

// Prepares directories by recursively creating target directory, setting its mode to
// 755 and also setting ownership if an user is provided
func prepareDirectories(targetDirectory string, owner, group int) {

}

// process processes listed TV shows by prompting user
func process(wd string, tvShows []TVShow, owner, group int) error {
	for _, tvShow := range tvShows {
		fmt.Println()

		prompt := promptui.Prompt{
			Label:     fmt.Sprintf("Process %s", tvShow.Name),
			IsConfirm: true,
			Default:   "y",
		}
		_, err := prompt.Run()
		if err != nil {
			if err.Error() == "^C" {
				return nil
			}
			continue
		}
		tvShowPath := path.Join(wd, tvShow.Name)
		os.Mkdir(tvShowPath, util.DirectoryMode)
		os.Chmod(tvShowPath, util.DirectoryMode)
		os.Chown(tvShowPath, owner, group)

		for _, season := range tvShow.Seasons {
			prompt := promptui.Prompt{
				Label:     season.Name,
				IsConfirm: true,
				Default:   "y",
			}
			_, err := prompt.Run()
			if err != nil {
				if err.Error() == "^C" {
					return nil
				}
				continue
			}

			seasonPath := path.Join(tvShowPath, season.Name)
			os.Mkdir(seasonPath, util.DirectoryMode)
			os.Chmod(seasonPath, util.DirectoryMode)
			os.Chown(seasonPath, owner, group)

			for _, episode := range season.Episodes {
				oldPath := path.Join(wd, episode.Basename)
				newPath := path.Join(seasonPath, episode.Fullname)
				os.Rename(oldPath, newPath)
				os.Chown(newPath, owner, group)
				os.Chmod(newPath, util.FileMode)

				fmt.Println(promptui.Styler(promptui.FGGreen)("✔"), episode.Fullname)
			}
		}
	}
	return nil
}

// FormatTVShowsCmd is the TV Shows-specific format command
var FormatTVShowsCmd = &cobra.Command{
	Use:   "tvshows <directory>",
	Short: "TV Shows-specific batch formatting",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		extensions, _ := cmd.Flags().GetStringArray("ext")
		tvShows, err := loadTVShows(media.WD, extensions)
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if err == nil {
			if len(tvShows) == 0 {
				fmt.Println(promptui.Styler(promptui.FGGreen)("✔"), "Nothing to process")
			} else {
				printAll(media.WD, tvShows)
				if !dryRun {
					process(media.WD, tvShows, media.UID, media.GID)
				}
			}
		}
		return err
	},
}
