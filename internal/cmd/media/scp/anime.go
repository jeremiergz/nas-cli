package scp

import (
	"github.com/spf13/cobra"
)

var (
	animeDesc = "Upload animes"
)

func newAnimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "animes <assets> <subpath>",
		Aliases: []string{"ani", "a"},
		Short:   animeDesc,
		Long:    animeDesc + ".",
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// animesDest := viper.GetString(config.KeySCPDestAnimesPath)
			// if animesDest == "" {
			// 	return fmt.Errorf("%s configuration entry is missing", config.KeySCPDestAnimesPath)
			// }

			// return process(cmd.Context(), cmd.OutOrStdout(), assets, animesDest, subpath)
			return nil
		},
	}

	cmd.MarkFlagDirname("assets")
	cmd.MarkFlagFilename("assets")

	return cmd
}
