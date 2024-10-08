package scp

import (
	"github.com/spf13/cobra"
)

var (
	tvShowDesc = "Upload TV shows"
)

func newTVShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tvshows <assets> <subpath>",
		Aliases: []string{"tv", "t"},
		Short:   tvShowDesc,
		Long:    tvShowDesc + ".",
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// tvShowsDest := viper.GetString(config.KeySCPDestTVShowsPath)
			// if tvShowsDest == "" {
			// 	return fmt.Errorf("%s configuration entry is missing", config.KeySCPDestTVShowsPath)
			// }

			// return process(cmd.Context(), cmd.OutOrStdout(), assets, tvShowsDest, subpath)
			return nil
		},
	}

	cmd.MarkFlagDirname("assets")
	cmd.MarkFlagFilename("assets")

	return cmd
}
