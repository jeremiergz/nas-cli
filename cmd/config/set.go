package config

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var SetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set configuration entry",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		if err := preflightChecks(key); err != nil {
			return err
		}

		viper.Set(key, value)

		tempFilePath := path.Join(WD, configFileNameWithoutDot)
		destFilePath := path.Join(WD, configFileName)

		err := viper.WriteConfigAs(tempFilePath)
		if err != nil {
			return fmt.Errorf("could not write configuration: %s", err)
		}
		err = os.Rename(tempFilePath, destFilePath)
		if err != nil {
			return fmt.Errorf("could not rename temporary file: %s", err)
		}

		return nil
	},
}
