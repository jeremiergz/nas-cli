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
	ConfigKeySCPGroup      string = "scp.group"
	ConfigKeySCPMovies     string = "scp.movies"
	ConfigKeySCPTVShows    string = "scp.tvshows"
	ConfigKeySCPUser       string = "scp.user"
	ConfigKeySSHHost       string = "ssh.host"
	ConfigKeySSHKnownHosts string = "ssh.knownhosts"
	ConfigKeySSHPort       string = "ssh.port"
	ConfigKeySSHPrivateKey string = "ssh.privatekey"
	ConfigKeySSHUsername   string = "ssh.username"
)

var (
	ConfigKeys = []string{
		ConfigKeyNASDomain,
		ConfigKeySCPAnimes,
		ConfigKeySCPGroup,
		ConfigKeySCPMovies,
		ConfigKeySCPTVShows,
		ConfigKeySCPUser,
		ConfigKeySSHHost,
		ConfigKeySSHKnownHosts,
		ConfigKeySSHPort,
		ConfigKeySSHPrivateKey,
		ConfigKeySSHUsername,
	}
	Dir      string
	FileName string = ".nascliconfig"
)

func init() {
	Dir, _ = os.UserHomeDir()
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
