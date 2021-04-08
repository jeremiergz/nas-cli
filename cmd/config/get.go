package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var GetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get configuration entry value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		if err := preflightChecks(key); err != nil {
			return err
		}

		value := viper.GetString(key)
		fmt.Println(value)

		return nil
	},
}
