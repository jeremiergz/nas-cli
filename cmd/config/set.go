package config

import (
	"github.com/jeremiergz/nas-cli/util/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var SetCmd = &cobra.Command{
	Use:       "set <key> <value>",
	Short:     "Set configuration entry",
	Args:      cobra.ExactArgs(2),
	ValidArgs: config.ConfigKeys,
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		viper.Set(key, value)
		return config.Save()
	},
}
