package config

import (
	"fmt"
	"os"
	"os/user"
	"path"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// DirectoryMode is the default mode to apply to directories
	DirectoryMode os.FileMode = 0755

	// ExecutableMode is the default mode for executable files
	ExecutableMode os.FileMode = 0755

	// FileMode is the default mode to apply to files
	FileMode os.FileMode = 0644

	KeyNASDomain     string = "nas.domain"
	KeySCPAnimes     string = "scp.animes"
	KeySCPGroup      string = "scp.group"
	KeySCPMovies     string = "scp.movies"
	KeySCPTVShows    string = "scp.tvshows"
	KeySCPUser       string = "scp.user"
	KeySSHHost       string = "ssh.host"
	KeySSHKnownHosts string = "ssh.knownhosts"
	KeySSHPort       string = "ssh.port"
	KeySSHPrivateKey string = "ssh.privatekey"
	KeySSHUsername   string = "ssh.username"
)

var (
	// GID is the processed files group to set
	GID int

	// Configuration directory
	Dir string

	// Configuration file name
	FileName string = ".nascliconfig"

	// Configuration keys
	Keys = []string{
		KeyNASDomain,
		KeySCPAnimes,
		KeySCPGroup,
		KeySCPMovies,
		KeySCPTVShows,
		KeySCPUser,
		KeySSHHost,
		KeySSHKnownHosts,
		KeySSHPort,
		KeySSHPrivateKey,
		KeySSHUsername,
	}

	// UID is the processed files owner to set
	UID int

	// WD is the working directory's absolute path
	WD string
)

func init() {
	Dir, _ = os.UserHomeDir()
	cobra.OnInitialize(func() {
		viper.SetConfigName(FileName)
		viper.AddConfigPath(Dir)
		viper.SetConfigType("ini")
		viper.AutomaticEnv()
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				panic(err)
			}
		}

		nasDomain := viper.GetString(KeyNASDomain)
		viper.SetDefault(KeySSHHost, viper.GetString(KeyNASDomain))
		if nasDomain != "" && viper.GetString(KeySSHHost) == "" {
			viper.Set(KeySSHHost, nasDomain)
		}

		homedir, _ := os.UserHomeDir()
		defaultKnownHosts := path.Join(homedir, ".ssh", "known_hosts")
		viper.SetDefault(KeySSHKnownHosts, defaultKnownHosts)

		viper.SetDefault(KeySSHPort, "22")

		defaultPrivateKey := path.Join(homedir, ".ssh", "id_rsa")
		viper.SetDefault(KeySSHPrivateKey, defaultPrivateKey)

		currentUser, _ := user.Current()
		var defaultUsername string
		if currentUser != nil {
			defaultUsername = currentUser.Username
		} else {
			defaultUsername = os.Getenv("USER")
		}
		viper.SetDefault(KeySSHUsername, defaultUsername)

		err := Save()
		if err != nil {
			fmt.Println(promptui.Styler(promptui.FGRed)("✗"), err.Error())
			os.Exit(1)
		}
	})
}

func Save() error {
	tempFilePath := path.Join(Dir, fmt.Sprintf("%s%s", FileName, ".bak.ini"))
	destFilePath := path.Join(Dir, FileName)

	err := viper.WriteConfigAs(tempFilePath)
	if err != nil {
		return fmt.Errorf("could not write configuration: %s", err)
	}
	err = os.Rename(tempFilePath, destFilePath)
	if err != nil {
		return fmt.Errorf("could not rename temporary file: %s", err)
	}

	return nil
}
