package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.com/jeremiergz/nas-cli/cmd/completion"
	"gitlab.com/jeremiergz/nas-cli/cmd/media"
	"gitlab.com/jeremiergz/nas-cli/cmd/version"
)

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "override config file")
	RootCmd.AddCommand(completion.CompletionCmd)
	RootCmd.AddCommand(media.MediaCmd)
	RootCmd.AddCommand(version.VersionCmd)
}

func initConfig() {
	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.AddConfigPath(defaultConfigDir)
		viper.SetConfigName(defaultConfigFile)
	}
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var (
	configFile        string
	defaultConfigDir  = "$HOME"
	defaultConfigFile = ".nas-cli"

	// RootCmd adds all child commands and sets flags appropriately
	RootCmd = &cobra.Command{
		Use:   "nas-cli",
		Short: "CLI application for managing a NAS",
	}
)
