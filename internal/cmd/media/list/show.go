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
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/config"
	svc "github.com/jeremiergz/nas-cli/internal/service"
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

	if recursive {
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
		lw.AppendItem(show.Name)
		if recursive {
			for _, season := range show.Seasons {
				lw.Indent()
				lw.AppendItem(season.Name)
				lw.Indent()
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

type show struct {
	mu   sync.Mutex
	sftp *sftp.Client

	RemoteDir string
	Name      string
	Seasons   []*season
}

type season struct {
	Name     string
	Episodes []string
}

func (s *show) loadSeasons() error {
	showPath := filepath.Join(s.RemoteDir, s.Name)

	seasonEntries, err := s.sftp.ReadDir(showPath)
	if err != nil {
		return err
	}
	sortSeasons(seasonEntries)

	eg := errgroup.Group{}

	for _, seasonEntry := range seasonEntries {
		eg.Go(func() error {
			seasonPath := filepath.Join(showPath, seasonEntry.Name())

			episodeEntries, err := s.sftp.ReadDir(seasonPath)
			if err != nil {
				return err
			}
			sortFiles(episodeEntries)

			episodes := []string{}
			for _, episodeEntry := range episodeEntries {
				episodes = append(episodes, episodeEntry.Name())
			}

			s.mu.Lock()
			s.Seasons = append(s.Seasons, &season{
				Name:     seasonEntry.Name(),
				Episodes: episodes,
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
