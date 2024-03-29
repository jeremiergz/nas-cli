package format

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"

	"github.com/disiqueira/gotree/v3"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/config"
	"github.com/jeremiergz/nas-cli/model"
	consoleservice "github.com/jeremiergz/nas-cli/service/console"
	mediaservice "github.com/jeremiergz/nas-cli/service/media"
	"github.com/jeremiergz/nas-cli/util/ctxutil"
)

var (
	yes         bool
	tvShowNames []string
)

func newTVShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tvshows <directory>",
		Aliases: []string{"tv", "t"},
		Short:   "TV Shows batch formatting",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			consoleSvc := ctxutil.Singleton[*consoleservice.Service](ctx)
			mediaSvc := ctxutil.Singleton[*mediaservice.Service](ctx)

			tvShows, err := mediaSvc.LoadTVShows(config.WD, extensions, nil, nil, false)

			w := cmd.OutOrStdout()

			if len(tvShowNames) > 0 {
				if len(tvShowNames) != len(tvShows) {
					return fmt.Errorf("names must be provided for all TV shows")
				}
				for index, tvShowName := range tvShowNames {
					tvShows[index].Name = tvShowName
				}
			}

			if err != nil {
				return err
			}
			if len(tvShows) == 0 {
				consoleSvc.Success("Nothing to process")
			} else {
				printAllTVShows(w, config.WD, tvShows)
				if !dryRun {
					processTVShows(cmd.Context(), w, config.WD, tvShows, config.UID, config.GID, !yes)
				}
			}

			return nil
		},
	}

	cmd.MarkFlagDirname("directory")
	cmd.Flags().StringArrayVarP(&tvShowNames, "name", "n", nil, "override TV show name")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "automatic yes to prompts")

	return cmd
}

// Prints given TV shows as a tree
func printAllTVShows(w io.Writer, wd string, tvShows []*model.TVShow) {
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
	fmt.Fprintln(w, toPrint)
}

// Processes listed TV shows by prompting user
func processTVShows(ctx context.Context, w io.Writer, wd string, tvShows []*model.TVShow, owner, group int, ask bool) error {
	consoleSvc := ctxutil.Singleton[*consoleservice.Service](ctx)
	mediaSvc := ctxutil.Singleton[*mediaservice.Service](ctx)

	for _, tvShow := range tvShows {
		fmt.Fprintln(w)

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
		mediaSvc.PrepareDirectory(tvShowPath, owner, group)

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
			mediaSvc.PrepareDirectory(seasonPath, owner, group)

			for _, episode := range season.Episodes {
				oldPath := path.Join(wd, episode.Basename)
				newPath := path.Join(seasonPath, episode.Name())
				os.Rename(oldPath, newPath)
				os.Chown(newPath, owner, group)
				os.Chmod(newPath, config.FileMode)
				consoleSvc.Success(episode.Name())
			}
		}
	}

	return nil
}
