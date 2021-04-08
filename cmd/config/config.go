package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/jeremiergz/nas-cli/cmd/media/scp"
	"github.com/jeremiergz/nas-cli/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	configFileName           string = ".nascliconfig"
	configFileNameWithoutDot string = "nascliconfig"
)

var (
	allowedKeys = []string{
		"nas.domain",
	}

	WD string
)

func init() {
	allowedKeys = append(allowedKeys, scp.ConfigKeys...)

	viper.SetConfigName(configFileName)
	viper.AddConfigPath("$HOME")
	viper.SetConfigType("ini")
	Cmd.AddCommand(GetCmd)
	Cmd.AddCommand(ListCmd)
	Cmd.AddCommand(SetCmd)
}

func preflightChecks(key string) error {
	if !util.StringInSlice(key, allowedKeys) {
		return fmt.Errorf("%s is not a valid configuration entry\nallowed keys: %s", key, strings.Join(allowedKeys, ", "))
	}
	return nil
}

var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Configure application",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		WD, err = os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not find user home directory")
		}
		return nil
	},
}
