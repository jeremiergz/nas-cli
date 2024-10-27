package list

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

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
	movieDesc = "List movies"
)

func newMovieCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "movies [name]",
		Aliases: []string{"mov", "m"},
		Short:   movieDesc,
		Long:    movieDesc + ".",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			out := cmd.OutOrStdout()

			moviesFolders := viper.GetStringSlice(config.KeySCPDestMoviesPaths)
			if len(moviesFolders) == 0 {
				return fmt.Errorf("%s configuration entry is missing", config.KeySCPDestMoviesPaths)
			}

			var movieName string
			if len(args) > 0 {
				movieName = args[0]
			}

			folders, err := listFolders(ctx, moviesFolders)
			if err != nil {
				return err
			}

			if len(folders) == 0 {
				fmt.Fprintln(cmd.OutOrStderr(), "Nothing to list")
				return nil
			}

			err = processMovies(cmd.Context(), out, folders, movieName)
			if err != nil {
				return err
			}

			return nil
		},
	}
	return cmd
}

func processMovies(
	ctx context.Context,
	out io.Writer,
	folders map[string][]fs.FileInfo,
	nameFilter string,
) error {
	eg, _ := errgroup.WithContext(ctx)
	eg.SetLimit(cmdutil.MaxConcurrentGoroutines)

	movies := []*movie{}

	for destination, entries := range folders {
		for _, entry := range entries {
			if nameFilter != "" {
				if !strings.Contains(strings.ToLower(entry.Name()), strings.ToLower(nameFilter)) {
					return nil
				}
			}
			movies = append(movies, &movie{
				sftp:      svc.SFTP.Client,
				RemoteDir: destination,
				Name:      entry.Name(),
			})
		}
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	if recursive {
		for _, movie := range movies {
			eg.Go(func() error {
				err := movie.loadFiles()
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

	printMovies(out, folders, movies)

	return nil
}

func printMovies(out io.Writer, folders map[string][]fs.FileInfo, movies []*movie) {
	lw := cmdutil.NewListWriter()

	for folder := range folders {
		filesCount := len(folders[folder])
		lw.AppendItem(fmt.Sprintf("%s (%d result%s)",
			filepath.Clean(folder),
			filesCount,
			lo.Ternary(filesCount > 1, "s", ""),
		))
	}

	sortMovies(movies)

	lw.Indent()
	for _, movie := range movies {
		lw.AppendItem(movie.Name)
		if recursive {
			for _, file := range movie.Files {
				lw.Indent()
				lw.AppendItem(file)
				lw.UnIndent()
			}
		}
	}

	fmt.Fprintln(out, lw.Render())
}

type movie struct {
	sftp *sftp.Client

	RemoteDir string
	Name      string
	Files     []string
}

func (m *movie) loadFiles() error {
	moviePath := filepath.Join(m.RemoteDir, m.Name)

	movieEntries, err := m.sftp.ReadDir(moviePath)
	if err != nil {
		return err
	}

	sortFiles(movieEntries)

	for _, movieEntry := range movieEntries {
		m.Files = append(m.Files, movieEntry.Name())
	}

	return nil
}

func sortMovies(movies []*movie) {
	slices.SortFunc(movies, func(i, j *movie) int {
		return cmp.Compare(
			strings.ToLower(i.Name),
			strings.ToLower(j.Name),
		)
	})
}
