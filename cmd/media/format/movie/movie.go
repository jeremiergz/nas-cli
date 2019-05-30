package movie

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"

	"github.com/logrusorgru/aurora"
	"github.com/manifoldco/promptui"
	"gitlab.com/jeremiergz/nas-cli/util"
	"gitlab.com/jeremiergz/nas-cli/util/media"
	"gitlab.com/jeremiergz/nas-cli/util/openfaas"

	gotree "github.com/DiSiqueira/GoTree"
	"github.com/spf13/cobra"
)

// Movie is the type of data that will be formatted as a movie
type Movie struct {
	Basename  string
	Extension string
	Fullname  string
	Title     string
	Year      int
}

var movieFmtRegexp = regexp.MustCompile(`(^.+)\s\(([0-9]{4})\)\.(.+)$`)

// getFullname returns generated fullname from given parameters
func getFullname(title string, year int, extension string) string {
	return fmt.Sprintf("%s (%d).%s", title, year, extension)
}

// loadMovies lists movies in folder that must be processed
func loadMovies(wd string, extensions []string) ([]Movie, error) {
	toProcess := media.List(wd, extensions, movieFmtRegexp)

	jsonBody, _ := json.Marshal(toProcess)
	responseData, err := openfaas.InvokeFaaS(openfaas.ParseMediaTitle, jsonBody)
	if err != nil {
		return nil, err
	}

	var parsedJSONResponse []struct {
		Basename  string
		Container string
		Title     string
		Year      int
	}
	err = json.Unmarshal(responseData, &parsedJSONResponse)
	if err != nil {
		return nil, err
	}

	movies := []Movie{}
	for _, movie := range parsedJSONResponse {
		movies = append(movies, Movie{
			Basename:  movie.Basename,
			Extension: movie.Container,
			Fullname:  getFullname(movie.Title, movie.Year, movie.Container),
			Title:     movie.Title,
			Year:      movie.Year,
		})
	}

	return movies, nil
}

// printAll prints given movies array as a tree
func printAll(wd string, movies []Movie) {
	moviesTree := gotree.New(wd)
	for _, m := range movies {
		moviesTree.Add(fmt.Sprintf("%s  %s", m.Fullname, aurora.Gray(10, m.Basename)))
	}
	toPrint := moviesTree.Print()
	lastSpaceRegexp := regexp.MustCompile(`\s$`)
	toPrint = lastSpaceRegexp.ReplaceAllString(toPrint, "")
	fmt.Println(toPrint)
}

// process processes listed movies by prompting user
func process(wd string, movies []Movie, owner, group int) error {
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
			Default: m.Title,
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
		yearInput, err := prompt.Run()
		yearInt, err := strconv.Atoi(yearInput)
		if err != nil {
			if err.Error() == "^C" {
				return nil
			}
			continue
		}

		newMovieName := getFullname(titleInput, yearInt, m.Extension)

		currentFilepath := path.Join(wd, m.Basename)
		newFilepath := path.Join(wd, newMovieName)
		os.Rename(currentFilepath, newFilepath)
		os.Chown(newFilepath, owner, group)
		os.Chmod(newFilepath, util.FileMode)

		fmt.Println(promptui.Styler(promptui.FGGreen)("✔"), newMovieName)
	}
	return nil
}

// FormatMoviesCmd is the movies-specific format command
var FormatMoviesCmd = &cobra.Command{
	Use:   "movies <directory>",
	Short: "Movies-specific batch formatting",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		extensions, _ := cmd.Flags().GetStringArray("ext")
		movies, err := loadMovies(media.WD, extensions)
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if err == nil {
			if len(movies) == 0 {
				fmt.Println(promptui.Styler(promptui.FGGreen)("✔"), "Nothing to process")
			} else {
				printAll(media.WD, movies)
				if !dryRun {
					process(media.WD, movies, media.UID, media.GID)
				}
			}
		}
		return err
	},
}
