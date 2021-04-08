package config

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	configFileName           string = ".nascliconfig"
	configFileNameWithoutDot string = "nascliconfig"
)

var (
	configDir string
)

func init() {
	viper.SetConfigName(configFileName)
	viper.AddConfigPath("$HOME")
	viper.SetConfigType("ini")
	Cmd.AddCommand(GetCmd)
	Cmd.AddCommand(ListCmd)
	Cmd.AddCommand(SetCmd)
}

var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Configure application",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		configDir, err = os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not find user home directory")
		}
		return nil
	},
}
