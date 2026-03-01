package model

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/samber/lo"

	"github.com/jeremiergz/nas-cli/internal/model/image"
	"github.com/jeremiergz/nas-cli/internal/model/internal/parser"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	_ MediaFile = (*Episode)(nil)

	showFmtRegexp = regexp.MustCompile(`(^.+)(\s-\s)S\d+E\d+\.(.+)$`)
)

// Holds information about a show such as its name and seasons.
type Show struct {
	images        []*image.Image
	name          string
	seasons       []*Season
	seasonsCount  int
	episodesCount int
}

func Shows(wd string, extensions []string, recursive bool, subtitleExtension string, subtitleLangs []string, anyFiles bool) ([]*Show, error) {
	var selectedRegexp *regexp.Regexp
	if !anyFiles {
		selectedRegexp = showFmtRegexp
	}

	toProcess := fsutil.List(wd, extensions, selectedRegexp, recursive)
	shows := []*Show{}
	for _, path := range toProcess {
		basename := filepath.Base(path)
		e, err := parser.Parse(basename)
		e.Title = util.ToTitleCase(e.Title)

		if err == nil {
			var show *Show
			showIndex := findShowIndex(e.Title, shows)
			if showIndex == -1 {
				baseImages, err := listBaseImageFiles(wd, e.Title)
				if err != nil {
					return nil, fmt.Errorf("failed to list show images for %s: %w", e.Title, err)
				}

				seasonImages, err := listSeasonImageFiles(wd, e.Title)
				if err != nil {
					return nil, fmt.Errorf("failed to list show season images for %s: %w", e.Title, err)
				}

				show = &Show{
					images:  slices.Concat(baseImages, seasonImages),
					name:    e.Title,
					seasons: []*Season{},
				}
			} else {
				show = shows[showIndex]
			}
			seasonName := fmt.Sprintf("Season %d", e.Season)
			seasonIndex := findShowSeasonIndex(seasonName, show.Seasons())

			f, err := newFile(basename, e.Container, filepath.Join(wd, path))
			if err != nil {
				return nil, err
			}

			episode := Episode{
				file:  f,
				index: e.Episode,
			}

			var season *Season
			if seasonIndex == -1 {
				season = &Season{
					episodes: []*Episode{},
					index:    e.Season,
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
		} else {
			return nil, err
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

func (s *Show) Images() []*image.Image {
	return s.images
}

func (s *Show) ConvertImagesToRequirements() error {
	for i, img := range s.images {
		newPath, err := convertImageFileToRequirements(img.FilePath, img.Kind)
		if err != nil {
			return fmt.Errorf("failed to convert show %s image file %s: %w", img.Kind, img.FilePath, err)
		}
		s.images[i].FilePath = newPath
	}
	return nil
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
	return fmt.Sprintf("%s - S%02dE%02d",
		e.Season().Show().Name(),
		e.Season().Index(),
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
