package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/config"
)

func NewGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "get <key>",
		Short:     "Get configuration entry value",
		ValidArgs: config.OrderedKeys,
		Args: func(cmd *cobra.Command, args []string) error {
			err := cobra.ExactArgs(1)(cmd, args)
			if err != nil {
				return err
			}

			return cobra.OnlyValidArgs(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			value := viper.GetString(key)
			fmt.Println(value)

			return nil
		},
	}

	return cmd
}
