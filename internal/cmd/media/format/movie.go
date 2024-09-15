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
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	consolesvc "github.com/jeremiergz/nas-cli/internal/service/console"
	mediasvc "github.com/jeremiergz/nas-cli/internal/service/media"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/ctxutil"
)

func newMovieCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "movies <directory>",
		Aliases: []string{"movie", "m"},
		Short:   "Movies batch formatting",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			consoleSvc := ctxutil.Singleton[*consolesvc.Service](ctx)
			mediaSvc := ctxutil.Singleton[*mediasvc.Service](ctx)

			w := cmd.OutOrStdout()

			movies, err := mediaSvc.LoadMovies(config.WD, extensions)
			if err != nil {
				return err
			}
			if len(movies) == 0 {
				consoleSvc.Success("Nothing to process")
			} else {
				consoleSvc.PrintMovies(movies)
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

var (
	movieNameCaser = cases.Title(language.Und)
)

// Processes listed movies by prompting user.
func processMovies(ctx context.Context, w io.Writer, wd string, movies []*model.Movie, owner, group int) error {
	consoleSvc := ctxutil.Singleton[*consolesvc.Service](ctx)

	for _, m := range movies {
		fmt.Fprintln(w)
		// Ask if current movie must be processed.
		prompt := promptui.Prompt{
			Label:     fmt.Sprintf("Rename %s", m.Basename),
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
			Default: movieNameCaser.String(m.Title),
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
			Default: strconv.Itoa(m.Year),
		}
		yearInput, _ := prompt.Run()
		yearInt, err := strconv.Atoi(yearInput)
		if err != nil {
			if err.Error() == "^C" {
				return nil
			}
			continue
		}
		newMovieName := util.ToMovieName(titleInput, yearInt, m.Extension)
		currentFilepath := path.Join(wd, m.Basename)
		newFilepath := path.Join(wd, newMovieName)
		os.Rename(currentFilepath, newFilepath)
		os.Chown(newFilepath, owner, group)
		os.Chmod(newFilepath, config.FileMode)
		consoleSvc.Success(newMovieName)
	}

	return nil
}
