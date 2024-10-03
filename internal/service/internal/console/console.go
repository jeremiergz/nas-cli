package console

import (
	"fmt"
	"io"
	"path/filepath"
	"regexp"

	"github.com/cheggaaa/pb/v3/termutil"
	"github.com/disiqueira/gotree/v3"
	"github.com/manifoldco/promptui"
	"github.com/samber/lo"

	"github.com/jeremiergz/nas-cli/internal/model"
)

type Service struct {
	w io.Writer
}

func New(w io.Writer) *Service {
	return &Service{w}
}

// Pretty-prints given error message.
func (s *Service) Error(message string) {
	fmt.Fprintln(s.w, promptui.Styler(promptui.FGRed)("✗"), message)
}

// Retrieves the terminal width to use when printing on console.
func (s *Service) GetTerminalWidth() int {
	termWidth, err := termutil.TerminalWidth()
	defaultWidth := 100
	if err != nil {
		termWidth = defaultWidth
	}

	return termWidth
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
				seasonsTree.Add(fmt.Sprintf("%s  %s", episode.Name(), episode.Basename()))
			}
		}
	}
	toPrint := rootTree.Print()
	toPrint = lastSpaceRegexp.ReplaceAllString(toPrint, "")

	fmt.Fprintln(s.w, toPrint)
}
