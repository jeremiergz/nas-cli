package list

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/sftp"
	"github.com/pterm/pterm"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/config"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	animeDesc  = "List animes"
	tvShowDesc = "List TV shows"
)

func newAnimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "animes [name]",
		Aliases: []string{"ani", "a"},
		Short:   animeDesc,
		Long:    animeDesc + ".",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			out := cmd.OutOrStdout()

			animesFolders := viper.GetStringSlice(config.KeySCPDestAnimesPaths)
			if len(animesFolders) == 0 {
				return fmt.Errorf("%s configuration entry is missing", config.KeySCPDestAnimesPaths)
			}

			var animeName string
			if len(args) > 0 {
				animeName = args[0]
			}

			folders, err := listFolders(ctx, animesFolders)
			if err != nil {
				return err
			}

			if len(folders) == 0 {
				fmt.Fprintln(cmd.OutOrStderr(), "Nothing to list")
				return nil
			}

			err = processShows(ctx, out, folders, animeName)
			if err != nil {
				return err
			}

			return nil
		},
	}
	return cmd
}

func newTVShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tvshows [name]",
		Aliases: []string{"tv", "t"},
		Short:   tvShowDesc,
		Long:    tvShowDesc + ".",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			out := cmd.OutOrStdout()

			tvShowsFolders := viper.GetStringSlice(config.KeySCPDestTVShowsPaths)
			if len(tvShowsFolders) == 0 {
				return fmt.Errorf("%s configuration entry is missing", config.KeySCPDestTVShowsPaths)
			}

			var tvShowName string
			if len(args) > 0 {
				tvShowName = args[0]
			}

			folders, err := listFolders(ctx, tvShowsFolders)
			if err != nil {
				return err
			}

			if len(folders) == 0 {
				fmt.Fprintln(cmd.OutOrStderr(), "Nothing to list")
				return nil
			}

			err = processShows(ctx, out, folders, tvShowName)
			if err != nil {
				return err
			}

			return nil
		},
	}
	return cmd
}

func processShows(
	ctx context.Context,
	out io.Writer,
	folders map[string][]fs.FileInfo,
	nameFilter string,
) error {
	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(cmdutil.MaxConcurrentGoroutines)

	spinner, err := pterm.DefaultSpinner.Start("Loading information...")
	if err != nil {
		return fmt.Errorf("could not start spinner: %w", err)
	}

	shows := []*show{}
	showsGroupedByFolder := map[string][]*show{}

	for destination, entries := range folders {
		for _, entry := range entries {
			if nameFilter != "" {
				if !strings.Contains(strings.ToLower(entry.Name()), strings.ToLower(nameFilter)) {
					continue
				}
			}
			s := &show{
				sftp:      svc.SFTP.Client,
				RemoteDir: destination,
				Name:      entry.Name(),
			}
			showsGroupedByFolder[destination] = append(showsGroupedByFolder[destination], s)
			shows = append(shows, s)
		}
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	for _, show := range shows {
		eg.Go(func() error {
			err := show.loadSeasons()
			if err != nil {
				return err
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	if flagOnlyComplete || flagOnlyIncomplete || flagOnlyPartial {
		filterFlags := map[showState]bool{
			showStateComplete:   flagOnlyComplete,
			showStateIncomplete: flagOnlyIncomplete,
			showStatePartial:    flagOnlyPartial,
		}

		for folder, showGroup := range showsGroupedByFolder {
			filteredGroup := lo.Filter(showGroup, func(s *show, _ int) bool {
				return filterFlags[s.State]
			})
			showsGroupedByFolder[folder] = filteredGroup
		}
	}

	if err := spinner.Stop(); err != nil {
		return fmt.Errorf("could not stop spinner: %w", err)
	}

	printShows(out, showsGroupedByFolder)

	return nil
}

func printShows(out io.Writer, showsGroupedByFolder map[string][]*show) {
	lw := cmdutil.NewListWriter()

	shows := []*show{}

	for folder, showGroup := range showsGroupedByFolder {
		filesCount := len(showGroup)
		lw.AppendItem(fmt.Sprintf("%s (%d result%s)",
			filepath.Clean(folder),
			filesCount,
			lo.Ternary(filesCount > 1, "s", ""),
		))
		shows = append(shows, showGroup...)
	}

	sortShows(shows)

	lw.Indent()
	for _, show := range shows {
		var showName string
		switch show.State {
		case showStateComplete:
			showName = pterm.Green(show.Name)
		case showStatePartial:
			showName = pterm.Magenta(show.Name)
		case showStateIncomplete:
			showName = pterm.Red(show.Name)
		default:
			showName = show.Name
		}

		lw.AppendItem(showName)
		if flagExtended {
			for _, file := range show.Files {
				lw.Indent()
				lw.AppendItem(file)
				lw.UnIndent()
			}
			for _, season := range show.Seasons {
				lw.Indent()
				lw.AppendItem(season.Name)
				lw.Indent()
				for _, file := range season.Files {
					lw.AppendItem(file)
				}
				for _, episode := range season.Episodes {
					lw.AppendItem(episode)
				}
				lw.UnIndent()
				lw.UnIndent()
			}
		}
	}

	fmt.Fprintln(out, lw.Render())
}

type showState int

const (
	showStateUnknown    showState = iota
	showStateComplete             // Show file + "poster.jpg" + "background.jpg" + "Season<number>.jpg"
	showStateIncomplete           // Only the show file is present.
	showStatePartial              // Missing either "poster.jpg", "background.jpg" or "Season<number>.jpg".
)

type show struct {
	mu   sync.Mutex
	sftp *sftp.Client

	RemoteDir string
	Name      string
	Seasons   []*season
	Files     []string
	State     showState
}

type season struct {
	Name     string
	Episodes []string
	Files    []string
}

func (s *show) loadSeasons() error {
	showPath := filepath.Join(s.RemoteDir, s.Name)

	showEntries, err := s.sftp.ReadDir(showPath)
	if err != nil {
		return err
	}

	sortSeasons(showEntries)

	eg := errgroup.Group{}

	hasBackgroundImageFile := false
	hasPosterImageFile := false
	hasAllSeasonPosterImageFiles := true

	for _, showEntry := range showEntries {
		eg.Go(func() error {
			showEntryName := showEntry.Name()
			if showEntryName == "background.jpg" {
				hasBackgroundImageFile = true
				s.Files = append(s.Files, showEntryName)
				return nil
			}
			if showEntryName == "poster.jpg" {
				hasPosterImageFile = true
				s.Files = append(s.Files, showEntryName)
				return nil
			}

			if !strings.HasPrefix(showEntryName, "Season") {
				return nil
			}

			// Handle season directory now that we've established it is one.

			seasonPath := filepath.Join(showPath, showEntry.Name())

			seasonEntries, err := s.sftp.ReadDir(seasonPath)
			if err != nil {
				return fmt.Errorf("failed to read season directory %s: %w", seasonPath, err)
			}

			sortFiles(seasonEntries)

			episodes := []string{}
			seasonFiles := []string{}
			hasSeasonPosterImageFile := false
			for _, seasonEntry := range seasonEntries {
				seasonEntryName := seasonEntry.Name()
				// FIXME: Poster detection should be based on the exact season number.
				isSeasonPoster :=
					(strings.HasPrefix(seasonEntryName, "Season") && strings.HasSuffix(seasonEntryName, ".jpg")) ||
						seasonEntryName == "season-specials-poster.jpg"
				if isSeasonPoster {
					seasonFiles = append(seasonFiles, seasonEntryName)
					hasSeasonPosterImageFile = true
				} else if slices.Contains(util.AcceptedVideoExtensions, strings.ToLower(filepath.Ext(seasonEntryName)[1:])) {
					episodes = append(episodes, seasonEntry.Name())
				}
			}

			if !hasSeasonPosterImageFile {
				hasAllSeasonPosterImageFiles = false
			}

			s.mu.Lock()
			s.Seasons = append(s.Seasons, &season{
				Name:     showEntry.Name(),
				Episodes: episodes,
				Files:    seasonFiles,
			})
			s.mu.Unlock()
			return nil
		})
	}
	if err = eg.Wait(); err != nil {
		return err
	}

	slices.SortFunc(s.Seasons, func(i, j *season) int {
		return cmp.Compare(i.Name, j.Name)
	})

	if hasBackgroundImageFile && hasPosterImageFile && hasAllSeasonPosterImageFiles {
		s.State = showStateComplete
	} else if hasBackgroundImageFile || hasPosterImageFile || hasAllSeasonPosterImageFiles {
		s.State = showStatePartial
	} else {
		s.State = showStateIncomplete
	}

	return nil
}

func sortShows(shows []*show) {
	slices.SortFunc(shows, func(i, j *show) int {
		return cmp.Compare(
			strings.ToLower(i.Name),
			strings.ToLower(j.Name),
		)
	})
}

func sortSeasons(seasons []fs.FileInfo) {
	slices.SortFunc(seasons, func(i, j fs.FileInfo) int {
		a, _ := strconv.Atoi(strings.Replace(i.Name(), "Season ", "", 1))
		b, _ := strconv.Atoi(strings.Replace(j.Name(), "Season ", "", 1))

		return cmp.Compare(a, b)
	})
}
