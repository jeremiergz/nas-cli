package upload

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag/v2"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	movieDesc  = "Upload movies"
	flagSortBy SortBy
)

// Defines the sortBy enumeration type.
type SortBy enumflag.Flag

// Enumeration values for the SortBy type.
const (
	SortByYear SortBy = iota
	SortByName
)

// Maps enumeration values to their textual representations.
var SortByIDs = map[SortBy][]string{
	SortByYear: {"year", "y"},
	SortByName: {"name", "n"},
}

func newMovieCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "movies <assets>",
		Aliases: []string{"movie", "mov", "m"},
		Short:   movieDesc,
		Long:    movieDesc + ".",
		Args:    cobra.MaximumNArgs(0),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmdutil.DebugMode {
				fmt.Fprintf(cmd.OutOrStdout(), "%s PreRunE\n", cmd.CommandPath())
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			err := processMovies(cmd.Context(), cmd.OutOrStdout())
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.MarkFlagDirname("assets")
	cmd.MarkFlagFilename("assets")
	cmd.Flags().VarP(
		enumflag.New(&flagSortBy, "", SortByIDs, enumflag.EnumCaseInsensitive),
		"sort-by",
		"s",
		"sort results by: year|name",
	)
	cmd.RegisterFlagCompletionFunc("sort-by", sortByCompletion)

	return cmd
}

func processMovies(ctx context.Context, out io.Writer) error {
	movies, err := model.Movies(config.WD, []string{util.ExtensionMKV}, recursive)
	if err != nil {
		return err
	}

	if len(movies) == 0 {
		svc.Console.Success("Nothing to upload")
		return nil
	}

	switch flagSortBy {
	case SortByYear:
		model.SortMoviesByYear(movies)
	case SortByName:
		model.SortMoviesByName(movies)
	}

	var spinner *pterm.SpinnerPrinter

	uploads := make([]*upload, len(movies))
	for i, movie := range movies {
		movieDirname := strings.TrimSuffix(movie.FullName(), fmt.Sprintf(".%s", movie.Extension()))
		uploads[i] = &upload{
			File:        movie,
			Destination: filepath.Join(remoteDirWithLowestUsage, movieDirname, movie.Basename()),
			DisplayName: movie.FullName(),
		}
		if len(movie.Images()) > 0 {
			// Start spinner if not already started.
			if spinner == nil {
				if spinner, err = pterm.DefaultSpinner.Start("Processing images..."); err != nil {
					return fmt.Errorf("could not start spinner: %w", err)
				}
			}
			if err := movie.ConvertImagesToRequirements(); err != nil {
				return fmt.Errorf("failed to convert %s image files: %w", movie.FullName(), err)
			}
			uploads[i].ImageFiles = movie.Images()
		}
	}

	// Stop spinner if it was started to convert images.
	if spinner != nil {
		if err := spinner.Stop(); err != nil {
			return fmt.Errorf("could not stop spinner: %w", err)
		}
	}

	err = process(ctx, out, uploads, model.KindMovie)
	if err != nil {
		return err
	}

	return nil
}

func sortByCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"year\tsort by year",
		"name\tsort by name",
	}, cobra.ShellCompDirectiveDefault
}
