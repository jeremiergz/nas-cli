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
	"github.com/pterm/pterm"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"github.com/jeremiergz/nas-cli/internal/config"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/ctxutil"
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
	moviesGroupedByFolder := map[string][]*movie{}

	for destination, entries := range folders {
		moviesGroupedByFolder[destination] = []*movie{}
		for _, entry := range entries {
			if nameFilter != "" {
				if !strings.Contains(strings.ToLower(entry.Name()), strings.ToLower(nameFilter)) {
					continue
				}
			}
			m := &movie{
				sftp:      svc.SFTP.Client,
				RemoteDir: destination,
				Name:      entry.Name(),
			}
			moviesGroupedByFolder[destination] = append(moviesGroupedByFolder[destination], m)
			movies = append(movies, m)
		}
	}
	if err := eg.Wait(); err != nil {
		return err
	}

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

	if flagOnlyComplete || flagOnlyIncomplete || flagOnlyPartial {
		filterFlags := map[movieState]bool{
			movieStateComplete:   flagOnlyComplete,
			movieStateIncomplete: flagOnlyIncomplete,
			movieStatePartial:    flagOnlyPartial,
		}

		for folder, movieGroup := range moviesGroupedByFolder {
			filteredGroup := lo.Filter(movieGroup, func(m *movie, _ int) bool {
				return filterFlags[m.State]
			})
			moviesGroupedByFolder[folder] = filteredGroup
		}
	}

	spinner := ctxutil.Loader(ctx)
	if spinner != nil {
		if err := spinner.Stop(); err != nil {
			return fmt.Errorf("could not stop spinner: %w", err)
		}
	}

	printMovies(out, moviesGroupedByFolder)

	return nil
}

func printMovies(out io.Writer, moviesGroupedByFolder map[string][]*movie) {
	lw := cmdutil.NewListWriter()

	movies := []*movie{}

	for folder, movieGroup := range moviesGroupedByFolder {
		filesCount := len(movieGroup)
		lw.AppendItem(fmt.Sprintf("%s (%d result%s)",
			filepath.Clean(folder),
			filesCount,
			lo.Ternary(filesCount > 1, "s", ""),
		))
		movies = append(movies, movieGroup...)
	}

	sortMovies(movies)

	lw.Indent()
	for _, movie := range movies {
		var movieName string
		switch movie.State {
		case movieStateComplete:
			movieName = pterm.Green(movie.Name)
		case movieStatePartial:
			movieName = pterm.Magenta(movie.Name)
		case movieStateIncomplete:
			movieName = pterm.Red(movie.Name)
		default:
			movieName = movie.Name
		}

		lw.AppendItem(movieName)
		if flagExtended {
			for _, file := range movie.Files {
				lw.Indent()
				lw.AppendItem(file)
				lw.UnIndent()
			}
		}
	}

	fmt.Fprintln(out, lw.Render())
}

type movieState int

const (
	movieStateUnknown    movieState = iota
	movieStateComplete              // Movie file + "poster.jpg" + "background.jpg".
	movieStateIncomplete            // Only the movie file is present.
	movieStatePartial               // Missing either "poster.jpg" or "background.jpg".
)

type movie struct {
	sftp *sftp.Client

	RemoteDir string
	Name      string
	Files     []string
	State     movieState
}

func (m *movie) loadFiles() error {
	moviePath := filepath.Join(m.RemoteDir, m.Name)

	movieEntries, err := m.sftp.ReadDir(moviePath)
	if err != nil {
		return err
	}

	sortFiles(movieEntries)

	hasMovieFile := false
	hasBackgroundImageFile := false
	hasPosterImageFile := false

	for _, movieEntry := range movieEntries {
		movieEntryName := movieEntry.Name()
		if slices.Contains(util.AcceptedVideoExtensions, strings.ToLower(filepath.Ext(movieEntryName)[1:])) {
			hasMovieFile = true
		}
		if movieEntryName == "background.jpg" {
			hasBackgroundImageFile = true
		}
		if movieEntryName == "poster.jpg" {
			hasPosterImageFile = true
		}

		m.Files = append(m.Files, movieEntryName)
	}

	if hasMovieFile {
		if hasBackgroundImageFile && hasPosterImageFile {
			m.State = movieStateComplete
		} else if hasBackgroundImageFile || hasPosterImageFile {
			m.State = movieStatePartial
		} else {
			m.State = movieStateIncomplete
		}
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
