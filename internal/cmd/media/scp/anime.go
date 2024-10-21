package scp

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	animeDesc = "Upload animes"
)

func newAnimeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "animes <assets>",
		Aliases: []string{"anime", "ani", "a"},
		Short:   animeDesc,
		Long:    animeDesc + ".",
		Args:    cobra.MinimumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmdutil.DebugMode {
				fmt.Fprintf(cmd.OutOrStdout(), "%s PreRunE\n", cmd.CommandPath())
			}

			err := cmdutil.CallParentPersistentPreRunE(cmd, args)
			if err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("remoteDiskUsageStats", remoteDiskUsageStats)
			fmt.Println("remoteDirWithLowestUsage", remoteDirWithLowestUsage)

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
