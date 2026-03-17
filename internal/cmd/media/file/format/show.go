package format

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/media"
	"github.com/jeremiergz/nas-cli/internal/prompt"
)

var (
	showDesc  = "Batch format shows"
	showNames []string
)

func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "shows <directory>",
		Aliases: []string{"show", "s"},
		Short:   showDesc,
		Long:    showDesc + ".",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shows, err := media.ListShows(config.WD, extensions, false, "", nil, false)
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
				pterm.Success.Println("Nothing to process")
				return nil
			}

			media.PrintShows(config.WD, shows)

			if !dryRun {
				var p prompt.Prompter
				if yes {
					p = prompt.NewAuto()
				} else {
					p = prompt.NewInteractive()
				}
				return processShows(cmd.Context(), cmd.OutOrStdout(), config.WD, shows, config.UID, config.GID, p)
			}

			return nil
		},
	}

	cmd.MarkFlagDirname("directory")
	cmd.Flags().StringArrayVarP(&showNames, "name", "n", nil, "override show name")

	return cmd
}

// Processes listed shows using the given prompter for user interaction.
func processShows(_ context.Context, w io.Writer, wd string, shows []*media.Show, owner, group int, p prompt.Prompter) error {
	for _, show := range shows {
		fmt.Fprintln(w)

		confirmed, err := p.Confirm(fmt.Sprintf("Process %s", show.Name()), true)
		if err != nil {
			return nil
		}
		if !confirmed {
			continue
		}

		for _, season := range show.Seasons() {
			confirmed, err := p.Confirm(season.Name(), true)
			if err != nil {
				return nil
			}
			if !confirmed {
				continue
			}

			for _, episode := range season.Episodes() {
				currentPath := filepath.Join(wd, episode.Basename())
				newPath := filepath.Join(wd, episode.FullName())

				if err := os.Rename(currentPath, newPath); err != nil {
					return fmt.Errorf("could not rename %s to %s: %w", currentPath, newPath, err)
				}

				os.Chown(newPath, owner, group)
				os.Chmod(newPath, config.FileMode)

				pterm.Success.Println(episode.FullName())
			}
		}
	}

	return nil
}
