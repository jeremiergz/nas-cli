package scp

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/config"
)

func newAnimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "animes <assets> <subpath>",
		Aliases: []string{"ani", "a"},
		Short:   "Animes uploading",
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			animesDest := viper.GetString(config.KeySCPDestAnimesPath)
			if animesDest == "" {
				return fmt.Errorf("%s configuration entry is missing", config.KeySCPDestAnimesPath)
			}

			return process(cmd.Context(), assets, animesDest, subpath)
		},
	}

	cmd.MarkFlagDirname("assets")
	cmd.MarkFlagFilename("assets")

	return cmd
}
