package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.com/jeremiergz/nas-cli/cmd/completion"
	"gitlab.com/jeremiergz/nas-cli/cmd/info"
	"gitlab.com/jeremiergz/nas-cli/cmd/media"
	"gitlab.com/jeremiergz/nas-cli/cmd/version"
)

func init() {
	cobra.OnInitialize(initConfig)
	Cmd.AddCommand(completion.Cmd)
	Cmd.AddCommand(info.Cmd)
	Cmd.AddCommand(media.Cmd)
	Cmd.AddCommand(version.Cmd)
}

func initConfig() {
	viper.AutomaticEnv()
}

var (
	// Cmd adds all child commands and sets global flags
	Cmd = &cobra.Command{
		Use:   "nas-cli",
		Short: "CLI application for managing my NAS",
	}
)
