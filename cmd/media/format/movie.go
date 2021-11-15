package format

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/jeremiergz/nas-cli/util"
	"github.com/jeremiergz/nas-cli/util/console"
	"github.com/jeremiergz/nas-cli/util/media"
	"github.com/manifoldco/promptui"

	gotree "github.com/DiSiqueira/GoTree"
	"github.com/spf13/cobra"
)

func init() {
	MovieCmd.MarkFlagDirname("directory")
	MovieCmd.MarkFlagFilename("directory")
}

var movieFmtRegexp = regexp.MustCompile(`(^.+)\s\(([0-9]{4})\)\.(.+)$`)

// loadMovies lists movies in folder that must be processed
func loadMovies(wd string, extensions []string) ([]media.Movie, error) {
	toProcess := media.List(wd, extensions, movieFmtRegexp)
	movies := []media.Movie{}
	for _, basename := range toProcess {
		m, err := media.ParseTitle(basename)
		if err == nil {
			movies = append(movies, media.Movie{
				Basename:  basename,
				Extension: m.Container,
				Fullname:  media.ToMovieName(m.Title, m.Year, m.Container),
				Title:     m.Title,
				Year:      m.Year,
			})
		} else {
			return nil, err
		}
	}
	return movies, nil
}

// printAllMovies prints given movies array as a tree
func printAllMovies(wd string, movies []media.Movie) {
	moviesTree := gotree.New(wd)
	for _, m := range movies {
		moviesTree.Add(fmt.Sprintf("%s  %s", m.Fullname, m.Basename))
	}
	toPrint := moviesTree.Print()
	lastSpaceRegexp := regexp.MustCompile(`\s$`)
	toPrint = lastSpaceRegexp.ReplaceAllString(toPrint, "")
	fmt.Println(toPrint)
}

// processMovies processes listed movies by prompting user
func processMovies(wd string, movies []media.Movie, owner, group int) error {
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
		newMovieName := media.ToMovieName(titleInput, yearInt, m.Extension)
		currentFilepath := path.Join(wd, m.Basename)
		newFilepath := path.Join(wd, newMovieName)
		os.Rename(currentFilepath, newFilepath)
		os.Chown(newFilepath, owner, group)
		os.Chmod(newFilepath, util.FileMode)
		console.Success(newMovieName)
	}
	return nil
}

var MovieCmd = &cobra.Command{
	Use:   "movies <directory>",
	Short: "Movies batch formatting",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		extensions, _ := cmd.Flags().GetStringArray("ext")
		movies, err := loadMovies(media.WD, extensions)
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if err != nil {
			return err
		}
		if len(movies) == 0 {
			console.Success("Nothing to process")
		} else {
			printAllMovies(media.WD, movies)
			if !dryRun {
				err := processMovies(media.WD, movies, media.UID, media.GID)
				if err != nil {
					return err
				}
			}
		}

		return nil
	},
}
