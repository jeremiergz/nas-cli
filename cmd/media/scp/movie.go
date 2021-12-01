package scp

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/util/config"
)

func NewMovieCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "movies <assets> <subpath>",
		Aliases: []string{"mov", "m"},
		Short:   "Movies uploading",
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			moviesDest := viper.GetString(config.ConfigKeySCPMovies)
			if moviesDest == "" {
				return fmt.Errorf("%s configuration entry is missing", config.ConfigKeySCPMovies)
			}

			return process(moviesDest, subpath)
		},
	}

	cmd.MarkFlagDirname("assets")
	cmd.MarkFlagFilename("assets")

	return cmd
}
