package config

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/internal/config"
)

func newSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "set <key> <value>",
		Short:     "Set configuration entry",
		ValidArgs: config.OrderedKeys,
		Args: func(cmd *cobra.Command, args []string) error {
			err := cobra.ExactArgs(2)(cmd, args)
			if err != nil {
				return err
			}

			err = cobra.OnlyValidArgs(cmd, []string{args[0]})
			if err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]
			viper.Set(key, value)

			return config.Save()
		},
	}

	return cmd
}
