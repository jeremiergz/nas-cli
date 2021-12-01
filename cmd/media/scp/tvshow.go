package scp

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/util/config"
)

func NewTVShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tvshows <assets> <subpath>",
		Aliases: []string{"tv", "t"},
		Short:   "TV Shows uploading",
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			tvShowsDest := viper.GetString(config.ConfigKeySCPTVShows)
			if tvShowsDest == "" {
				return fmt.Errorf("%s configuration entry is missing", config.ConfigKeySCPTVShows)
			}

			return process(tvShowsDest, subpath)
		},
	}

	cmd.MarkFlagDirname("assets")
	cmd.MarkFlagFilename("assets")

	return cmd
}
