package scp

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/model"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	movieDesc = "Upload movies"
)

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
			movies, err := model.Movies(config.WD, []string{util.ExtensionMKV}, recursive)
			if err != nil {
				return err
			}

			if len(movies) == 0 {
				svc.Console.Success("Nothing to upload")
				return nil
			}

			uploads := make([]*upload, len(movies))
			for i, movie := range movies {
				movieDirname := strings.TrimSuffix(movie.FullName(), fmt.Sprintf(".%s", movie.Extension()))
				uploads[i] = &upload{
					File:        movie,
					Destination: filepath.Join(remoteDirWithLowestUsage, movieDirname, movie.Basename()),
					DisplayName: movie.FullName(),
				}
			}

			err = process(cmd.Context(), cmd.OutOrStdout(), uploads, model.KindMovie)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.MarkFlagDirname("assets")
	cmd.MarkFlagFilename("assets")

	return cmd
}
