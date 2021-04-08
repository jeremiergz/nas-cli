package config

import (
	"fmt"

	configutil "github.com/jeremiergz/nas-cli/util/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var GetCmd = &cobra.Command{
	Use:       "get <key>",
	Short:     "Get configuration entry value",
	Args:      cobra.ExactArgs(1),
	ValidArgs: configutil.ConfigKeys,
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		value := viper.GetString(key)
		fmt.Println(value)

		return nil
	},
}
