package scp

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	tvShowDesc = "Upload TV shows"
)

func newTVShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tvshows <assets>",
		Aliases: []string{"tvshow", "tv", "t"},
		Short:   tvShowDesc,
		Long:    tvShowDesc + ".",
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
