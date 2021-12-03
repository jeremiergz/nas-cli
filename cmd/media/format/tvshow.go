package format

import (
	"fmt"
	"os"
	"path"
	"regexp"

	"github.com/disiqueira/gotree/v3"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/util"
	"github.com/jeremiergz/nas-cli/util/console"
	"github.com/jeremiergz/nas-cli/util/media"
)

// Prints given TV shows as a tree
func printAllTVShows(wd string, tvShows []*media.TVShow) {
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

// Processes listed TV shows by prompting user
func processTVShows(wd string, tvShows []*media.TVShow, owner, group int, ask bool) error {
	for _, tvShow := range tvShows {
		fmt.Println()

		var err error
		if ask {
			prompt := promptui.Prompt{
				Label:     fmt.Sprintf("Process %s", tvShow.Name),
				IsConfirm: true,
				Default:   "y",
			}
			_, err = prompt.Run()
		}
		if err != nil {
			if err.Error() == "^C" {
				return nil
			}
			continue
		}

		tvShowPath := path.Join(wd, tvShow.Name)
		media.PrepareDirectory(tvShowPath, owner, group)

		for _, season := range tvShow.Seasons {
			if ask {
				prompt := promptui.Prompt{
					Label:     season.Name,
					IsConfirm: true,
					Default:   "y",
				}
				_, err = prompt.Run()
			}
			if err != nil {
				if err.Error() == "^C" {
					return nil
				}
				continue
			}

			seasonPath := path.Join(tvShowPath, season.Name)
			media.PrepareDirectory(seasonPath, owner, group)

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

func NewTVShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tvshows <directory>",
		Aliases: []string{"tv", "t"},
		Short:   "TV Shows batch formatting",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			extensions, _ := cmd.Flags().GetStringArray("ext")
			tvShowNames, _ := cmd.Flags().GetStringArray("name")
			yes, _ := cmd.Flags().GetBool("yes")

			tvShows, err := media.LoadTVShows(media.WD, extensions, nil, nil, false)

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
				printAllTVShows(media.WD, tvShows)
				if !dryRun {
					processTVShows(media.WD, tvShows, media.UID, media.GID, !yes)
				}
			}

			return nil
		},
	}

	cmd.MarkFlagDirname("directory")
	cmd.Flags().StringArrayP("name", "n", nil, "override TV show name")
	cmd.Flags().BoolP("yes", "y", false, "automatic yes to prompts")

	return cmd
}
