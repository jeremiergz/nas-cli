package scp

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/config"
)

func NewMovieCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "movies <assets> <subpath>",
		Aliases: []string{"mov", "m"},
		Short:   "Movies uploading",
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			moviesDest := viper.GetString(config.KeySCPDestMoviesPath)
			if moviesDest == "" {
				return fmt.Errorf("%s configuration entry is missing", config.KeySCPDestMoviesPath)
			}

			return process(cmd.Context(), assets, moviesDest, subpath)
		},
	}

	cmd.MarkFlagDirname("assets")
	cmd.MarkFlagFilename("assets")

	return cmd
}
