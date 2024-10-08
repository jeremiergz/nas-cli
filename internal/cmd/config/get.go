package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/internal/config"
)

var (
	getDesc = "Get configuration entry value"
)

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "get <key>",
		Short:     getDesc,
		Long:      getDesc + ".",
		ValidArgs: config.OrderedKeys,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			value := viper.GetString(key)
			fmt.Fprintln(cmd.OutOrStdout(), value)

			return nil
		},
	}

	return cmd
}
