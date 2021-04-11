package config

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/viper"
)

const (
	ConfigKeyNASDomain     string = "nas.domain"
	ConfigKeySCPAnimes     string = "scp.animes"
	ConfigKeySCPMovies     string = "scp.movies"
	ConfigKeySCPTVShows    string = "scp.tvshows"
	ConfigKeySSHHost       string = "ssh.host"
	ConfigKeySSHKnownHosts string = "ssh.knownhosts"
	ConfigKeySSHPort       string = "ssh.port"
	ConfigKeySSHPrivateKey string = "ssh.privatekey"
	ConfigKeySSHUsername   string = "ssh.username"
	FileName               string = ".nascliconfig"
)

var (
	configDir  string
	ConfigKeys = []string{
		ConfigKeyNASDomain,
		ConfigKeySCPAnimes,
		ConfigKeySCPMovies,
		ConfigKeySCPTVShows,
		ConfigKeySSHHost,
		ConfigKeySSHKnownHosts,
		ConfigKeySSHPort,
		ConfigKeySSHPrivateKey,
		ConfigKeySSHUsername,
	}
)

func init() {
	configDir, _ = os.UserHomeDir()
}

func Save() error {
	tempFilePath := path.Join(configDir, fmt.Sprintf("%s%s", FileName, ".bak.ini"))
	destFilePath := path.Join(configDir, FileName)

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
