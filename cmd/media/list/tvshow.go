package list

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/util/config"
)

func NewTVShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tvshows [name]",
		Aliases: []string{"tv", "t"},
		Short:   "TV Shows listing",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tvShowsDest := viper.GetString(config.ConfigKeySCPTVShows)
			if tvShowsDest == "" {
				return fmt.Errorf("%s configuration entry is missing", config.ConfigKeySCPTVShows)
			}

			var tvShowName string
			if len(args) > 0 {
				tvShowName = args[0]
			}

			return process(tvShowsDest, true, tvShowName)
		},
	}

	cmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "find files and folders recursively")

	return cmd
}