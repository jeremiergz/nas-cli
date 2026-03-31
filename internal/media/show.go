package media

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/pterm/pterm"
	"github.com/samber/lo"

	"github.com/jeremiergz/nas-cli/internal/image"
	"github.com/jeremiergz/nas-cli/internal/media/internal/parser"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	_ MediaFile = (*Episode)(nil)
)

// Holds information about a show such as its name and seasons.
type Show struct {
	images        []*image.Image
	name          string
	seasons       []*Season
	seasonsCount  int
	episodesCount int
}

// Lists shows in given folder. Useful when name is in expected format and can be parsed with a simple regexp.
//
// Result can be filtered by extensions.
func ListShows(wd string, extensions []string, recursive bool) ([]*Show, error) {
	return listShowsWithParser(wd, extensions, recursive, parseShowWithRegexp)
}

// Parses shows in given folder. Useful when name is not in expected format and requires a more complex parsing logic.
//
// Result can be filtered by extensions.
func ParseShows(wd string, extensions []string, recursive bool) ([]*Show, error) {
	return listShowsWithParser(wd, extensions, recursive, parseShowWithParser)
}

func (s *Show) Images() []*image.Image {
	return s.images
}

func (s *Show) Name() string {
	return s.name
}

func (s *Show) SetName(name string) {
	s.name = name
}

func (s *Show) Seasons() []*Season {
	return s.seasons
}

func (s *Show) SeasonsCount() int {
	return s.seasonsCount
}

func (s *Show) EpisodesCount() int {
	return s.episodesCount
}

// Holds information about a season such as its index and episodes.
type Season struct {
	name     string
	index    int
	show     *Show
	episodes []*Episode
}

func (s *Season) Name() string {
	return s.name
}

func (s *Season) Index() int {
	return s.index
}

func (s *Season) Show() *Show {
	return s.show
}

func (s *Season) Episodes() []*Episode {
	return s.episodes
}

// Holds information about an episode such as its index and subtitles.
type Episode struct {
	*file

	index  int
	season *Season
}

func (e *Episode) Index() int {
	return e.index
}

func (e *Episode) Name() string {
	width := max(numberOfDigits(e.Index()), 2)
	return fmt.Sprintf("%s - S%02dE%0*d",
		e.Season().Show().Name(),
		e.Season().Index(),
		width,
		e.Index(),
	)
}

func (e *Episode) FullName() string {
	return fmt.Sprintf("%s.%s",
		e.Name(),
		e.Extension(),
	)
}

func (e *Episode) Season() *Season {
	return e.season
}

func (e *Episode) PosterImageFilePath() *string {
	return nil
}

// Finds season index in seasons array.
func findShowSeasonIndex(name string, seasons []*Season) int {
	seasonIndex := -1
	for i, season := range seasons {
		if season.Name() == name {
			seasonIndex = i
			continue
		}
	}

	return seasonIndex
}

// Finds Show index in Shows array.
func findShowIndex(name string, shows []*Show) int {
	showIndex := -1
	for i, show := range shows {
		if show.Name() == name {
			showIndex = i
			continue
		}
	}

	return showIndex
}

var imageFileSeasonNumberRegexp = regexp.MustCompile(`\.[sS]\d{2,}$`)

func listSeasonImageFiles(dir, referenceName string) (imageFiles []*image.Image, err error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Keep only image files to reduce the number of iterations later on.
	files = slices.DeleteFunc(files, func(file os.DirEntry) bool {
		fileExtension := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Name()), "."))
		return !slices.Contains(image.ValidExtensions, fileExtension)
	})

	// Now, keep only files that end with .01, .02, etc. as they are the ones that can be associated with a season.
	files = slices.DeleteFunc(files, func(file os.DirEntry) bool {
		seasonNumberPart := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		return !imageFileSeasonNumberRegexp.MatchString(seasonNumberPart)
	})

	for _, file := range files {
		filePath := filepath.Join(".", file.Name())
		fileName := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		hasReferenceName := strings.HasPrefix(fileName, referenceName)

		if hasReferenceName {
			seasonNumberStr := strings.TrimPrefix(fileName, fmt.Sprintf("%s.", referenceName))
			seasonNumber, err := strconv.Atoi(seasonNumberStr[1:])
			if err != nil {
				return nil, fmt.Errorf("failed to parse season number for %s: %w", fileName, err)
			}

			imageFileNamePrefix := lo.Ternary(
				seasonNumber == 0,
				"season-specials-poster",
				fmt.Sprintf("Season%02d", seasonNumber),
			)
			imageFileName := fmt.Sprintf("Season %d/%s", seasonNumber, imageFileNamePrefix)
			imageFiles = append(imageFiles, image.New(imageFileName, filePath, image.KindPoster))
		}
	}

	return imageFiles, nil
}

// Prints given shows as a tree.
func PrintShows(wd string, shows []*Show) {
	lw := cmdutil.NewListWriter()
	showsCount := len(shows)

	lw.AppendItem(
		fmt.Sprintf(
			"%s (%d %s)",
			wd,
			showsCount,
			lo.Ternary(showsCount <= 1, "show", "shows"),
		),
	)

	lw.Indent()
	for _, show := range shows {
		lw.AppendItem(
			fmt.Sprintf(
				"%s (%d %s / %d %s)",
				show.Name(),
				show.SeasonsCount(),
				lo.Ternary(show.SeasonsCount() <= 1, "season", "seasons"),
				show.EpisodesCount(),
				lo.Ternary(show.EpisodesCount() <= 1, "episode", "episodes"),
			),
		)

		lw.Indent()
		for _, season := range show.Seasons() {
			episodesCount := len(season.Episodes())
			episodeStr := "episodes"
			if episodesCount == 1 {
				episodeStr = "episode"
			}
			lw.AppendItem(
				fmt.Sprintf(
					"%s (%d %s)",
					season.Name(),
					episodesCount,
					episodeStr,
				),
			)
			lw.Indent()
			for _, episode := range season.Episodes() {
				lw.AppendItem(fmt.Sprintf(
					"%s  <-  %s",
					episode.FullName(),
					pterm.Gray(episode.Basename()),
				),
				)
			}
			lw.UnIndent()
		}
		lw.UnIndent()
	}

	pterm.Println(lw.Render())
}

var showParsingRegexp = regexp.MustCompile(`^(?<name>.+)\s-\sS(?<season>\d{2})E(?<episode>\d{2,4})\.(?<extension>.{3})$`)

func parseShowWithRegexp(basename string) (name string, seasonNumber, episodeNumber int, extension string, err error) {
	matches := showParsingRegexp.FindStringSubmatch(basename)
	if len(matches) != 5 {
		return "", 0, 0, "", errors.New("filename does not match expected format")
	}

	name = matches[1]
	seasonNumber, _ = strconv.Atoi(matches[2])
	episodeNumber, _ = strconv.Atoi(matches[3])
	extension = matches[4]

	return name, seasonNumber, episodeNumber, extension, nil
}

func parseShowWithParser(basename string) (name string, seasonNumber, episodeNumber int, extension string, err error) {
	show, err := parser.Parse(basename)
	if err != nil {
		return "", 0, 0, "", fmt.Errorf("failed to parse show %s: %w", basename, err)
	}

	show.Title = util.ToTitleCase(show.Title)

	return show.Title, show.Season, show.Episode, show.Container, nil
}

func listShowsWithParser(
	wd string,
	extensions []string,
	recursive bool,
	parser func(basename string) (name string, seasonNumber, episodeNumber int, extension string, err error),
) ([]*Show, error) {
	toProcess := fsutil.List(wd, extensions, nil, recursive)
	shows := []*Show{}
	for _, path := range toProcess {
		basename := filepath.Base(path)

		name, seasonNumber, episodeNumber, extension, err := parser(basename)
		if err != nil {
			return nil, fmt.Errorf("failed to parse show %s: %w", basename, err)
		}

		var show *Show
		showIndex := findShowIndex(name, shows)
		if showIndex == -1 {
			baseImages, err := listBaseImageFiles(wd, name)
			if err != nil {
				return nil, fmt.Errorf("failed to list show images for %s: %w", name, err)
			}

			seasonImages, err := listSeasonImageFiles(wd, name)
			if err != nil {
				return nil, fmt.Errorf("failed to list show season images for %s: %w", name, err)
			}

			show = &Show{
				images:  slices.Concat(baseImages, seasonImages),
				name:    name,
				seasons: []*Season{},
			}
		} else {
			show = shows[showIndex]
		}
		seasonName := fmt.Sprintf("Season %d", seasonNumber)
		seasonIndex := findShowSeasonIndex(seasonName, show.Seasons())

		f, err := newFile(basename, extension, filepath.Join(wd, path))
		if err != nil {
			return nil, err
		}

		episode := Episode{
			file:  f,
			index: episodeNumber,
		}

		var season *Season
		if seasonIndex == -1 {
			season = &Season{
				episodes: []*Episode{},
				index:    seasonNumber,
				name:     seasonName,
				show:     show,
			}
			episode.season = season
			season.episodes = append(season.episodes, &episode)
			show.seasons = append(show.seasons, season)
		} else {
			season := show.seasons[seasonIndex]
			episode.season = season
			season.episodes = append(season.episodes, &episode)
		}

		for _, season := range show.seasons {
			slices.SortFunc(season.episodes, func(i, j *Episode) int {
				return cmp.Compare(i.index, j.index)
			})
		}
		slices.SortFunc(show.seasons, func(i, j *Season) int {
			return cmp.Compare(i.index, j.index)
		})

		if showIndex == -1 {
			shows = append(shows, show)
		}
	}

	for _, show := range shows {
		show.seasonsCount = len(show.seasons)
		for _, season := range show.seasons {
			show.episodesCount += len(season.episodes)
		}
	}

	return shows, nil
}

func numberOfDigits(n int) int {
	return len(strconv.Itoa(int(math.Abs(float64(n)))))
}
