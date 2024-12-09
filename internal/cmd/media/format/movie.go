package format

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	svc "github.com/jeremiergz/nas-cli/internal/service"
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
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			movies, err := model.Movies(config.WD, extensions, false)
			if err != nil {
				return err
			}

			if len(movies) == 0 {
				svc.Console.Success("Nothing to process")
				return nil
			}

			svc.Console.PrintMovies(config.WD, movies)

			if !dryRun {
				err := processMovies(cmd.Context(), cmd.OutOrStdout(), config.WD, movies, config.UID, config.GID)
				if err != nil {
					return err
				}

			}

			return nil
		},
	}

	cmd.MarkFlagDirname("directory")

	return cmd
}

// Processes listed movies by prompting user.
func processMovies(_ context.Context, w io.Writer, wd string, movies []*model.Movie, owner, group int) error {
	for _, m := range movies {
		fmt.Fprintln(w)
		// Ask if current movie must be processed.
		prompt := promptui.Prompt{
			Label:     fmt.Sprintf("Rename %s", m.Basename()),
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

		// Allow modification of parsed movie title.
		prompt = promptui.Prompt{
			Label:   "Name",
			Default: m.Name(),
		}
		titleInput, err := prompt.Run()
		if err != nil {
			if err.Error() == "^C" {
				return nil
			}
			continue
		}
		// Allow modification of parsed movie year.
		prompt = promptui.Prompt{
			Label:   "Year",
			Default: strconv.Itoa(m.Year()),
		}
		yearInput, _ := prompt.Run()
		yearInt, err := strconv.Atoi(yearInput)
		if err != nil {
			if err.Error() == "^C" {
				return nil
			}
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

		svc.Console.Success(m.FullName())
	}

	return nil
}
