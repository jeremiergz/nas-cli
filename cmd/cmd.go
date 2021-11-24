package cmd

import (
	"os"
	"os/user"
	"path"

	configutil "github.com/jeremiergz/nas-cli/util/config"
	"github.com/jeremiergz/nas-cli/util/console"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	viper.SetConfigName(configutil.FileName)
	viper.AddConfigPath("$HOME")
	viper.SetConfigType("ini")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			panic(err)
		}
	}

	nasDomain := viper.GetString(configutil.ConfigKeyNASDomain)
	viper.SetDefault(configutil.ConfigKeySSHHost, viper.GetString(configutil.ConfigKeyNASDomain))
	if nasDomain != "" && viper.GetString(configutil.ConfigKeySSHHost) == "" {
		viper.Set(configutil.ConfigKeySSHHost, nasDomain)
	}

	homedir, _ := os.UserHomeDir()
	defaultKnownHosts := path.Join(homedir, ".ssh", "known_hosts")
	viper.SetDefault(configutil.ConfigKeySSHKnownHosts, defaultKnownHosts)

	viper.SetDefault(configutil.ConfigKeySSHPort, "22")

	defaultPrivateKey := path.Join(homedir, ".ssh", "id_rsa")
	viper.SetDefault(configutil.ConfigKeySSHPrivateKey, defaultPrivateKey)

	currentUser, _ := user.Current()
	var defaultUsername string
	if currentUser != nil {
		defaultUsername = currentUser.Username
	} else {
		defaultUsername = os.Getenv("USER")
	}
	viper.SetDefault(configutil.ConfigKeySSHUsername, defaultUsername)

	err := configutil.Save()
	if err != nil {
		console.Error(err.Error())
		os.Exit(1)
	}
}

var Root = &cobra.Command{
	Use:   "nas-cli",
	Short: "CLI application for managing my NAS",
}
