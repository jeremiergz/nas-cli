package console

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/disiqueira/gotree/v3"
	"github.com/manifoldco/promptui"
	"github.com/samber/lo"

	"github.com/jeremiergz/nas-cli/internal/model"
)

type Service struct {
	w io.Writer
}

func New() *Service {
	return &Service{w: os.Stdout}
}

// Sets the output writer to use when printing on console.
func (s *Service) SetOutput(w io.Writer) {
	s.w = w
}

// Pretty-prints given error message.
func (s *Service) Error(message string) {
	fmt.Fprintln(s.w, promptui.Styler(promptui.FGRed)("✗"), message)
}

// Pretty-prints given info message.
func (s *Service) Info(message string) {
	fmt.Fprintln(s.w, promptui.Styler(promptui.FGYellow)("❯"), message)
}

// Pretty-prints given success message.
func (s *Service) Success(message string) {
	fmt.Fprintln(s.w, promptui.Styler(promptui.FGGreen)("✔"), message)
}

var (
	lastSpaceRegexp = regexp.MustCompile(`\s$`)
)

// Prints given files array as a tree.
func (s *Service) PrintFiles(wd string, files []*model.File) {
	tree := gotree.New(wd)
	for _, f := range files {
		tree.Add(f.Basename())
	}
	toPrint := tree.Print()
	toPrint = lastSpaceRegexp.ReplaceAllString(toPrint, "")

	fmt.Fprintln(s.w, toPrint)
}

// Prints given movies array as a tree.
func (s *Service) PrintMovies(wd string, movies []*model.Movie) {
	moviesTree := gotree.New(wd)
	for _, m := range movies {
		moviesTree.Add(fmt.Sprintf(
			"%s  %s",
			filepath.Join(m.FullName(), fmt.Sprintf("%s.%s", m.FullName(), m.Extension())),
			m.Basename(),
		))
	}
	toPrint := moviesTree.Print()
	toPrint = lastSpaceRegexp.ReplaceAllString(toPrint, "")

	fmt.Fprintln(s.w, toPrint)
}

// Prints given shows as a tree.
func (s *Service) PrintShows(wd string, shows []*model.Show) {
	rootTree := gotree.New(wd)
	for _, show := range shows {
		showTree := rootTree.Add(
			fmt.Sprintf(
				"%s (%d %s - %d %s)",
				show.Name(),
				show.SeasonsCount(),
				lo.Ternary(show.SeasonsCount() <= 1, "season", "seasons"),
				show.EpisodesCount(),
				lo.Ternary(show.EpisodesCount() <= 1, "episode", "episodes"),
			),
		)

		for _, season := range show.Seasons() {
			episodesCount := len(season.Episodes())
			episodeStr := "episodes"
			if episodesCount == 1 {
				episodeStr = "episode"
			}
			seasonsTree := showTree.Add(fmt.Sprintf("%s (%d %s)", season.Name(), episodesCount, episodeStr))
			for _, episode := range season.Episodes() {
				seasonsTree.Add(fmt.Sprintf("%s  %s", episode.FullName(), episode.Basename()))
			}
		}
	}
	toPrint := rootTree.Print()
	toPrint = lastSpaceRegexp.ReplaceAllString(toPrint, "")

	fmt.Fprintln(s.w, toPrint)
}

func (s *Service) AskConfirmation(label string, yesByDefault bool) bool {
	choices := "Y/n"
	if !yesByDefault {
		choices = "y/N"
	}

	fmt.Fprintf(s.w, "%s %s? [%s] ", s.Blue("?"), label, s.Gray(choices))

	var result bool
	for {
		r := bufio.NewReader(os.Stdin)
		str, err := r.ReadString('\n')
		if err != nil {
			fmt.Fprintln(s.w)
			result = false
			break
		}

		str = strings.TrimSpace(str)
		if str == "" {
			result = yesByDefault
			break
		}
		str = strings.ToLower(str)
		if str == "y" || str == "yes" {
			result = true
			break
		}
		if str == "n" || str == "no" {
			result = false
			break
		}
	}

	fmt.Printf("\033[1A\033[K%s %s  %s\n",
		lo.Ternary(result, s.Green("✔"), s.Red("✖")),
		label,
		s.Gray(lo.Ternary(result, "yes", "no")),
	)

	return result
}

var (
	reset = "\033[0m"
	blue  = "\033[34m"
	gray  = "\033[90m"
	green = "\033[32m"
	red   = "\033[31m"
)

func (s *Service) Blue(label string) string {
	return fmt.Sprintf("%s%s%s", blue, label, reset)
}

func (s *Service) Gray(label string) string {
	return fmt.Sprintf("%s%s%s", gray, label, reset)
}

func (s *Service) Green(label string) string {
	return fmt.Sprintf("%s%s%s", green, label, reset)
}

func (s *Service) Red(label string) string {
	return fmt.Sprintf("%s%s%s", red, label, reset)
}
