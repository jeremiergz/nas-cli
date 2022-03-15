package scp

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/config"
)

func NewTVShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tvshows <assets> <subpath>",
		Aliases: []string{"tv", "t"},
		Short:   "TV Shows uploading",
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			tvShowsDest := viper.GetString(config.KeySCPTVShows)
			if tvShowsDest == "" {
				return fmt.Errorf("%s configuration entry is missing", config.KeySCPTVShows)
			}

			return process(cmd.Context(), assets, tvShowsDest, subpath)
		},
	}

	cmd.MarkFlagDirname("assets")
	cmd.MarkFlagFilename("assets")

	return cmd
}
