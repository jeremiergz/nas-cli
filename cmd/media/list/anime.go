package list

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/util/config"
)

func NewAnimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "animes [name]",
		Aliases: []string{"ani", "a"},
		Short:   "Animes listing",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			animesDest := viper.GetString(config.ConfigKeySCPAnimes)
			if animesDest == "" {
				return fmt.Errorf("%s configuration entry is missing", config.ConfigKeySCPAnimes)
			}

			var animeName string
			if len(args) > 0 {
				animeName = args[0]
			}

			return process(animesDest, true, animeName)
		},
	}

	cmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "find files and folders recursively")

	return cmd
}
