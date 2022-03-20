package config

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/ini.v1"
)

const (
	// DirectoryMode is the default mode to apply to directories
	DirectoryMode os.FileMode = 0755

	// ExecutableMode is the default mode for executable files
	ExecutableMode os.FileMode = 0755

	// FileMode is the default mode to apply to files
	FileMode os.FileMode = 0644

	KeyNASFQDN             string = "nas.fqdn"
	KeySCPChownGroup       string = "scp.chown.group"
	KeySCPChownUser        string = "scp.chown.user"
	KeySCPDestAnimesPath   string = "scp.dest.animespath"
	KeySCPDestMoviesPath   string = "scp.dest.moviespath"
	KeySCPDestTVShowsPath  string = "scp.dest.tvshowspath"
	KeySSHClientKnownHosts string = "ssh.client.knownhosts"
	KeySSHClientPrivateKey string = "ssh.client.privatekey"
	KeySSHHost             string = "ssh.host"
	KeySSHPort             string = "ssh.port"
	KeySSHUser             string = "ssh.user"
)

var (
	// GID is the processed files group to set
	GID int

	// Configuration directory
	Dir string

	// Configuration file name
	FileName string = ".nascliconfig"

	// Configuration keys in INI file order
	Keys = []string{
		KeyNASFQDN,
		KeySCPChownGroup,
		KeySCPChownUser,
		KeySCPDestAnimesPath,
		KeySCPDestMoviesPath,
		KeySCPDestTVShowsPath,
		KeySSHHost,
		KeySSHPort,
		KeySSHUser,
		KeySSHClientKnownHosts,
		KeySSHClientPrivateKey,
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

		nasDomain := viper.GetString(KeyNASFQDN)
		viper.SetDefault(KeyNASFQDN, "localhost")

		viper.SetDefault(KeySCPChownGroup, "media")
		viper.SetDefault(KeySCPChownUser, "media")

		viper.SetDefault(KeySCPDestAnimesPath, "")
		viper.SetDefault(KeySCPDestMoviesPath, "")
		viper.SetDefault(KeySCPDestTVShowsPath, "")

		sshHost := viper.GetString(KeySSHHost)
		viper.SetDefault(KeySSHHost, "localhost")
		if nasDomain != "" && sshHost == "" {
			viper.Set(KeySSHHost, nasDomain)
		}

		homedir, _ := os.UserHomeDir()
		defaultKnownHosts := path.Join(homedir, ".ssh", "known_hosts")
		viper.SetDefault(KeySSHClientKnownHosts, defaultKnownHosts)

		viper.SetDefault(KeySSHPort, "22")

		defaultPrivateKey := path.Join(homedir, ".ssh", "id_rsa")
		viper.SetDefault(KeySSHClientPrivateKey, defaultPrivateKey)

		currentUser, _ := user.Current()
		var defaultUsername string
		if currentUser != nil {
			defaultUsername = currentUser.Username
		} else {
			defaultUsername = os.Getenv("USER")
		}
		viper.SetDefault(KeySSHUser, defaultUsername)

		err := Save()
		if err != nil {
			fmt.Println(promptui.Styler(promptui.FGRed)("✗"), err.Error())
			os.Exit(1)
		}
	})
}

func Save() error {
	configFilePath := path.Join(Dir, FileName)

	cfg := ini.Empty()
	for _, key := range Keys {
		lastSep := strings.LastIndex(key, ".")
		sectionName := key[:(lastSep)]
		keyName := key[(lastSep + 1):]
		if sectionName == "default" {
			sectionName = ""
		}
		cfg.Section(sectionName).Key(keyName).SetValue(viper.GetString(key))
	}

	file, err := os.Create(configFilePath)
	if err != nil {
		return fmt.Errorf("could not open file for write: %v", err)
	}
	defer file.Close()

	_, err = cfg.WriteTo(file)

	if err != nil {
		return fmt.Errorf("could not write configuration: %v", err)
	}

	return nil
}
