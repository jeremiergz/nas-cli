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

var MovieCmd = &cobra.Command{
	Use:   "movies <assets> <subpath>",
	Short: "Movies uploading",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		moviesDest := viper.GetString(ConfigKeyMovies)
		if moviesDest == "" {
			return fmt.Errorf("%s configuration entry is missing", ConfigKeyMovies)
		}
		return process(moviesDest, subpath)
	},
}
