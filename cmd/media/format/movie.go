package format

import (
	"context"
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/disiqueira/gotree/v3"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/config"
	"github.com/jeremiergz/nas-cli/model"
	"github.com/jeremiergz/nas-cli/service"
	"github.com/jeremiergz/nas-cli/util"
)

var movieFmtRegexp = regexp.MustCompile(`(^.+)\s\(([0-9]{4})\)\.(.+)$`)

// Lists movies in folder that must be processed
func loadMovies(ctx context.Context, wd string, extensions []string) ([]model.Movie, error) {
	mediaSvc := ctx.Value(util.ContextKeyMedia).(*service.MediaService)

	toProcess := mediaSvc.List(wd, extensions, movieFmtRegexp)
	movies := []model.Movie{}
	for _, basename := range toProcess {
		m, err := mediaSvc.ParseTitle(basename)
		if err == nil {
			movies = append(movies, model.Movie{
				Basename:  basename,
				Extension: m.Container,
				Fullname:  util.ToMovieName(m.Title, m.Year, m.Container),
				Title:     m.Title,
				Year:      m.Year,
			})
		} else {
			return nil, err
		}
	}

	return movies, nil
}

// Prints given movies array as a tree
func printAllMovies(wd string, movies []model.Movie) {
	moviesTree := gotree.New(wd)
	for _, m := range movies {
		moviesTree.Add(fmt.Sprintf("%s  %s", m.Fullname, m.Basename))
	}
	toPrint := moviesTree.Print()
	lastSpaceRegexp := regexp.MustCompile(`\s$`)
	toPrint = lastSpaceRegexp.ReplaceAllString(toPrint, "")
	fmt.Println(toPrint)
}

// Processes listed movies by prompting user
func processMovies(ctx context.Context, wd string, movies []model.Movie, owner, group int) error {
	consoleSvc := ctx.Value(util.ContextKeyConsole).(*service.ConsoleService)

	for _, m := range movies {
		fmt.Println()
		// Ask if current movie must be processed
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
		// Allow modification of parsed movie title
		prompt = promptui.Prompt{
			Label:   "Name",
			Default: strings.Title(m.Title),
		}
		titleInput, err := prompt.Run()
		if err != nil {
			if err.Error() == "^C" {
				return nil
			}
			continue
		}
		// Allow modification of parsed movie year
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

func NewMovieCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "movies <directory>",
		Aliases: []string{"mov", "m"},
		Short:   "Movies batch formatting",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			consoleSvc := cmd.Context().Value(util.ContextKeyConsole).(*service.ConsoleService)

			extensions, _ := cmd.Flags().GetStringArray("ext")
			movies, err := loadMovies(cmd.Context(), config.WD, extensions)
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			if err != nil {
				return err
			}
			if len(movies) == 0 {
				consoleSvc.Success("Nothing to process")
			} else {
				printAllMovies(config.WD, movies)
				if !dryRun {
					err := processMovies(cmd.Context(), config.WD, movies, config.UID, config.GID)
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
