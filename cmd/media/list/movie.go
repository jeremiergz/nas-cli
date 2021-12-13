package list

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/util/config"
)

func NewMovieCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "movies",
		Aliases: []string{"mov", "m"},
		Short:   "Movies listing",
		Args:    cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			moviesDest := viper.GetString(config.ConfigKeySCPMovies)
			if moviesDest == "" {
				return fmt.Errorf("%s configuration entry is missing", config.ConfigKeySCPMovies)
			}

			return process(moviesDest, false)
		},
	}

	return cmd
}
