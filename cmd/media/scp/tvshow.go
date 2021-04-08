package scp

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	Cmd.MarkFlagDirname("assets")
	Cmd.MarkFlagFilename("assets")
}

var TVShowCmd = &cobra.Command{
	Use:   "tvshows <assets> <subpath>",
	Short: "TV Shows uploading",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		tvShowsDest := viper.GetString(ConfigKeyTVShows)
		if tvShowsDest == "" {
			return fmt.Errorf("%s configuration entry is missing", ConfigKeyTVShows)
		}
		return process(tvShowsDest, subpath)
	},
}
