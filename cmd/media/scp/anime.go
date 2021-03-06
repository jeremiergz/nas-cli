package scp

import (
	"fmt"

	"github.com/jeremiergz/nas-cli/util/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	Cmd.MarkFlagDirname("assets")
	Cmd.MarkFlagFilename("assets")
}

var AnimeCmd = &cobra.Command{
	Use:   "animes <assets> <subpath>",
	Short: "Animes uploading",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		animesDest := viper.GetString(config.ConfigKeySCPAnimes)
		if animesDest == "" {
			return fmt.Errorf("%s configuration entry is missing", config.ConfigKeySCPAnimes)
		}

		return process(animesDest, subpath)
	},
}
