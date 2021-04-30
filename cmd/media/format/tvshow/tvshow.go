package tvshow

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	gotree "github.com/DiSiqueira/GoTree"
	"github.com/jeremiergz/nas-cli/util"
	"github.com/jeremiergz/nas-cli/util/console"
	"github.com/jeremiergz/nas-cli/util/media"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.MarkFlagDirname("directory")
	Cmd.MarkFlagFilename("directory")
	Cmd.Flags().StringArrayP("name", "n", nil, "override TV show name")
}

var tvShowFmtRegexp = regexp.MustCompile(`(^.+)(\s-\s)S\d+E\d+\.(.+)$`)

// findSeasonIndex in seasons array
func findSeasonIndex(name string, seasons []*media.Season) int {
	seasonIndex := -1
	for i, season := range seasons {
		if season.Name == name {
			seasonIndex = i
			continue
		}
	}
	return seasonIndex
}

// findTVShowIndex in TV Shows array
func findTVShowIndex(name string, tvShows []*media.TVShow) int {
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
func loadTVShows(wd string, extensions []string) ([]*media.TVShow, error) {
	toProcess := media.List(wd, extensions, tvShowFmtRegexp)
	tvShows := []*media.TVShow{}
	for _, basename := range toProcess {
		e, err := media.ParseTitle(basename)
		e.Title = strings.Title(e.Title)
		if err == nil {
			var tvShow *media.TVShow
			tvShowIndex := findTVShowIndex(e.Title, tvShows)
			if tvShowIndex == -1 {
				tvShow = &media.TVShow{
					Name:    e.Title,
					Seasons: []*media.Season{},
				}
			} else {
				tvShow = tvShows[tvShowIndex]
			}
			seasonName := media.ToSeasonName(e.Season)
			seasonIndex := findSeasonIndex(seasonName, tvShow.Seasons)
			episode := media.Episode{
				Basename:  basename,
				Extension: e.Container,
				Index:     e.Episode,
			}
			var season *media.Season
			if seasonIndex == -1 {
				season = &media.Season{
					Episodes: []*media.Episode{},
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

// printAll prints given TV shows as a tree
func printAll(wd string, tvShows []*media.TVShow) {
	rootTree := gotree.New(wd)
	for _, tvShow := range tvShows {
		tvShowTree := rootTree.Add(tvShow.Name)
		for _, season := range tvShow.Seasons {
			filesCount := len(season.Episodes)
			fileString := "files"
			if filesCount == 1 {
				fileString = "file"
			}
			seasonsTree := tvShowTree.Add(fmt.Sprintf("%s (%d %s)", season.Name, filesCount, fileString))
			for _, episode := range season.Episodes {
				seasonsTree.Add(fmt.Sprintf("%s  %s", episode.Name(), episode.Basename))
			}
		}
	}
	toPrint := rootTree.Print()
	lastSpaceRegexp := regexp.MustCompile(`\s$`)
	toPrint = lastSpaceRegexp.ReplaceAllString(toPrint, "")
	fmt.Println(toPrint)
}

// prepareDirectory by creating target directory, setting its mode to 755 and setting ownership
func prepareDirectory(targetDirectory string, owner, group int) {
	os.Mkdir(targetDirectory, util.DirectoryMode)
	os.Chmod(targetDirectory, util.DirectoryMode)
	os.Chown(targetDirectory, owner, group)
}

// process processes listed TV shows by prompting user
func process(wd string, tvShows []*media.TVShow, owner, group int) error {
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
		prepareDirectory(tvShowPath, owner, group)
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
			prepareDirectory(seasonPath, owner, group)
			for _, episode := range season.Episodes {
				oldPath := path.Join(wd, episode.Basename)
				newPath := path.Join(seasonPath, episode.Name())
				os.Rename(oldPath, newPath)
				os.Chown(newPath, owner, group)
				os.Chmod(newPath, util.FileMode)
				console.Success(episode.Name())
			}
		}
	}
	return nil
}

var Cmd = &cobra.Command{
	Use:   "tvshows <directory>",
	Short: "TV Shows batch formatting",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		extensions, _ := cmd.Flags().GetStringArray("ext")
		tvShowNames, _ := cmd.Flags().GetStringArray("name")
		tvShows, err := loadTVShows(media.WD, extensions)

		if len(tvShowNames) > 0 {
			if len(tvShowNames) != len(tvShows) {
				return fmt.Errorf("names must be provided for all TV shows")
			}
			for index, tvShowName := range tvShowNames {
				tvShows[index].Name = tvShowName
			}
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if err != nil {
			return err
		}
		if len(tvShows) == 0 {
			console.Success("Nothing to process")
		} else {
			printAll(media.WD, tvShows)
			if !dryRun {
				process(media.WD, tvShows, media.UID, media.GID)
			}
		}
		return nil
	},
}
