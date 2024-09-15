package format

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	consolesvc "github.com/jeremiergz/nas-cli/internal/service/console"
	mediasvc "github.com/jeremiergz/nas-cli/internal/service/media"
	"github.com/jeremiergz/nas-cli/internal/util/ctxutil"
)

var (
	yes       bool
	showNames []string
)

func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "shows <directory>",
		Aliases: []string{"show", "s"},
		Short:   "Shows batch formatting",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			consoleSvc := ctxutil.Singleton[*consolesvc.Service](ctx)
			mediaSvc := ctxutil.Singleton[*mediasvc.Service](ctx)

			shows, err := mediaSvc.LoadShows(config.WD, extensions, nil, nil, false)

			w := cmd.OutOrStdout()

			if len(showNames) > 0 {
				if len(showNames) != len(shows) {
					return fmt.Errorf("names must be provided for all shows")
				}
				for index, showName := range showNames {
					shows[index].Name = showName
				}
			}

			if err != nil {
				return err
			}
			if len(shows) == 0 {
				consoleSvc.Success("Nothing to process")
			} else {
				consoleSvc.PrintShows(shows)
				if !dryRun {
					processShows(cmd.Context(), w, config.WD, shows, config.UID, config.GID, !yes)
				}
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
func processShows(ctx context.Context, w io.Writer, wd string, shows []*model.Show, owner, group int, ask bool) error {
	consoleSvc := ctxutil.Singleton[*consolesvc.Service](ctx)
	mediaSvc := ctxutil.Singleton[*mediasvc.Service](ctx)

	for _, show := range shows {
		fmt.Fprintln(w)

		var err error
		if ask {
			prompt := promptui.Prompt{
				Label:     fmt.Sprintf("Process %s", show.Name),
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

		showPath := path.Join(wd, show.Name)
		mediaSvc.PrepareDirectory(showPath, owner, group)

		for _, season := range show.Seasons {
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

			seasonPath := path.Join(showPath, season.Name)
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
