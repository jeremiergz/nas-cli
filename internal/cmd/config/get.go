package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/internal/config"
)

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "get <key>",
		Short:     "Get configuration entry value",
		ValidArgs: config.OrderedKeys,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			w := cmd.OutOrStdout()

			value := viper.GetString(key)
			fmt.Fprintln(w, value)

			return nil
		},
	}

	return cmd
}
