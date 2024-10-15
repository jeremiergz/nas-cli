package format

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	showDesc  = "Batch format shows"
	yes       bool
	showNames []string
)

func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "shows <directory>",
		Aliases: []string{"show", "s"},
		Short:   showDesc,
		Long:    showDesc + ".",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shows, err := model.Shows(config.WD, extensions, "", nil, false)
			if err != nil {
				return err
			}

			if len(showNames) > 0 {
				if len(showNames) != len(shows) {
					return fmt.Errorf("names must be provided for all shows")
				}
				for index, showName := range showNames {
					shows[index].SetName(showName)
				}
			}

			if len(shows) == 0 {
				svc.Console.Success("Nothing to process")
				return nil
			}

			svc.Console.PrintShows(config.WD, shows)

			if !dryRun {
				processShows(cmd.Context(), cmd.OutOrStdout(), config.WD, shows, config.UID, config.GID, !yes)
			}

			return nil
		},
	}

	cmd.MarkFlagDirname("directory")
	cmd.Flags().StringArrayVarP(&showNames, "name", "n", nil, "override show name")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "automatic yes to prompts")

	return cmd
}

// Processes listed shows by prompting user.
func processShows(_ context.Context, w io.Writer, wd string, shows []*model.Show, owner, group int, ask bool) error {
	for _, show := range shows {
		fmt.Fprintln(w)

		var err error
		if ask {
			prompt := promptui.Prompt{
				Label:     fmt.Sprintf("Process %s", show.Name()),
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

		showPath := filepath.Join(wd, show.Name())
		fsutil.PrepareDir(showPath, owner, group)

		for _, season := range show.Seasons() {
			if ask {
				prompt := promptui.Prompt{
					Label:     season.Name(),
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

			seasonPath := filepath.Join(showPath, season.Name())
			fsutil.PrepareDir(seasonPath, owner, group)

			for _, episode := range season.Episodes() {
				oldPath := filepath.Join(wd, episode.Basename())
				newPath := filepath.Join(seasonPath, episode.FullName())
				os.Rename(oldPath, newPath)
				os.Chown(newPath, owner, group)
				os.Chmod(newPath, config.FileMode)
				svc.Console.Success(episode.FullName())
			}
		}
	}

	return nil
}
