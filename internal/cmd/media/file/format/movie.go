package format

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/media"
	"github.com/jeremiergz/nas-cli/internal/prompt"
)

var (
	movieDesc = "Batch format movies"
)

func newMovieCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "movies <directory>",
		Aliases: []string{"movie", "m"},
		Short:   movieDesc,
		Long:    movieDesc + ".",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			movies, err := media.ListMovies(config.WD, extensions, false)
			if err != nil {
				return err
			}

			if len(movies) == 0 {
				pterm.Success.Println("Nothing to process")
				return nil
			}

			media.PrintMovies(config.WD, movies)

			if !dryRun {
				var p prompt.Prompter
				if yes {
					p = prompt.NewAuto()
				} else {
					p = prompt.NewInteractive()
				}
				return processMovies(cmd.Context(), cmd.OutOrStdout(), config.WD, movies, config.UID, config.GID, p)
			}

			return nil
		},
	}

	cmd.MarkFlagDirname("directory")

	return cmd
}

// Processes listed movies using the given prompter for user interaction.
func processMovies(_ context.Context, w io.Writer, wd string, movies []*media.Movie, owner, group int, p prompt.Prompter) error {
	for _, m := range movies {
		fmt.Fprintln(w)

		// Ask if current movie must be processed.
		confirmed, err := p.Confirm(fmt.Sprintf("Rename %s", m.Basename()), true)
		if err != nil {
			return nil
		}
		if !confirmed {
			continue
		}

		// Allow modification of parsed movie title.
		titleInput, err := p.Input("Name", m.Name())
		if err != nil {
			return nil
		}

		// Allow modification of parsed movie year.
		yearInput, err := p.Input("Year", strconv.Itoa(m.Year()))
		if err != nil {
			return nil
		}
		yearInt, err := strconv.Atoi(yearInput)
		if err != nil {
			continue
		}
		m.SetName(titleInput)
		m.SetYear(yearInt)

		currentPath := filepath.Join(wd, m.Basename())
		newPath := filepath.Join(wd, m.FullName())

		if err := os.Rename(currentPath, newPath); err != nil {
			return fmt.Errorf("could not rename %s to %s: %w", currentPath, newPath, err)
		}

		os.Chown(newPath, owner, group)
		os.Chmod(newPath, config.FileMode)

		pterm.Success.Println(m.FullName())
	}

	return nil
}
