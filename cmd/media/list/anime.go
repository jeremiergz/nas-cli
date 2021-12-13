package list

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/util/config"
)

func NewAnimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "animes",
		Aliases: []string{"ani", "a"},
		Short:   "Animes listing",
		Args:    cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			animesDest := viper.GetString(config.ConfigKeySCPAnimes)
			if animesDest == "" {
				return fmt.Errorf("%s configuration entry is missing", config.ConfigKeySCPAnimes)
			}

			return process(animesDest, true)
		},
	}

	return cmd
}
