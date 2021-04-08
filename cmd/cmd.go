package cmd

import (
	"github.com/jeremiergz/nas-cli/cmd/completion"
	"github.com/jeremiergz/nas-cli/cmd/config"
	"github.com/jeremiergz/nas-cli/cmd/info"
	"github.com/jeremiergz/nas-cli/cmd/media"
	"github.com/jeremiergz/nas-cli/cmd/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cobra.OnInitialize(initConfig)
	Cmd.AddCommand(completion.Cmd)
	Cmd.AddCommand(config.Cmd)
	Cmd.AddCommand(info.Cmd)
	Cmd.AddCommand(media.Cmd)
	Cmd.AddCommand(version.Cmd)
}

func initConfig() {
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(err)
		}
	}
}

var Cmd = &cobra.Command{
	Use:   "nas-cli",
	Short: "CLI application for managing my NAS",
}
