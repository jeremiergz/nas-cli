package format

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	consolesvc "github.com/jeremiergz/nas-cli/internal/service/console"
	"github.com/jeremiergz/nas-cli/internal/util/ctxutil"
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
			ctx := cmd.Context()
			consoleSvc := ctxutil.Singleton[*consolesvc.Service](ctx)

			w := cmd.OutOrStdout()

			movies, err := model.Movies(config.WD, extensions)
			if err != nil {
				return err
			}
			if len(movies) == 0 {
				consoleSvc.Success("Nothing to process")
			} else {
				consoleSvc.PrintMovies(config.WD, movies)
				if !dryRun {
					err := processMovies(cmd.Context(), w, config.WD, movies, config.UID, config.GID)
					if err != nil {
						return err
					}
				}
			}

			return nil
		},
	}

	cmd.MarkFlagDirname("directory")

	return cmd
}

// Processes listed movies by prompting user.
func processMovies(ctx context.Context, w io.Writer, wd string, movies []*model.Movie, owner, group int) error {
	consoleSvc := ctxutil.Singleton[*consolesvc.Service](ctx)

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

		newDir := path.Join(wd, m.FullName())
		err = os.MkdirAll(newDir, config.DirectoryMode)
		if err != nil {
			return fmt.Errorf("could not create directory %s: %w", newDir, err)
		}

		currentFilepath := path.Join(wd, m.Basename())
		newFilepath := path.Join(newDir, fmt.Sprintf("%s.%s", m.FullName(), m.Extension()))
		err = os.Rename(currentFilepath, newFilepath)
		if err != nil {
			return fmt.Errorf("could not rename %s to %s: %w", currentFilepath, newFilepath, err)
		}

		os.Chown(newFilepath, owner, group)
		os.Chmod(newFilepath, config.FileMode)

		consoleSvc.Success(m.FullName())
	}

	return nil
}
