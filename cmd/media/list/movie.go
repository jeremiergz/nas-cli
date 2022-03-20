package list

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/config"
)

func NewMovieCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "movies [name]",
		Aliases: []string{"mov", "m"},
		Short:   "Movies listing",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			moviesDest := viper.GetString(config.KeySCPDestMoviesPath)
			if moviesDest == "" {
				return fmt.Errorf("%s configuration entry is missing", config.KeySCPDestMoviesPath)
			}

			var movieName string
			if len(args) > 0 {
				movieName = args[0]
			}

			return process(cmd.Context(), moviesDest, false, movieName)
		},
	}

	return cmd
}
