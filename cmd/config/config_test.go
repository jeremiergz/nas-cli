package config

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeremiergz/nas-cli/cmd"
	"github.com/jeremiergz/nas-cli/util/config"
	"github.com/jeremiergz/nas-cli/util/test"
	"github.com/spf13/viper"
)

var (
	testConfigFile     string = ".nascliconfig.test"
	testConfigFilePath string
)

func init() {
	home, _ := os.UserHomeDir()
	config.FileName = testConfigFile
	testConfigFilePath = path.Join(home, testConfigFile)
	cmd.Root.AddCommand(Cmd)
}

func cleanup() {
	os.Remove(testConfigFilePath)
}

func TestConfigGetCmd(t *testing.T) {
	t.Cleanup(cleanup)

	_, output := test.ExecuteCommand(t, cmd.Root, []string{"config", "get", config.ConfigKeySSHUsername})
	currentUser, _ := user.Current()

	test.AssertEquals(t, currentUser.Username, output)
}

func TestConfigListCmd(t *testing.T) {
	t.Cleanup(cleanup)

	_, output := test.ExecuteCommand(t, cmd.Root, []string{"config", "list"})

	for _, key := range viper.AllKeys() {
		test.AssertContains(t, fmt.Sprintf("%s=", key), output)
	}
}

func TestConfigSetCmd(t *testing.T) {
	t.Cleanup(cleanup)

	tests := []struct {
		key   string
		value string
	}{
		{config.ConfigKeyNASDomain, "nas.test.local"},
		{config.ConfigKeySCPAnimes, path.Join(os.TempDir(), "animes")},
		{config.ConfigKeySCPGroup, "test"},
		{config.ConfigKeySCPMovies, path.Join(os.TempDir(), "movies")},
		{config.ConfigKeySCPTVShows, path.Join(os.TempDir(), "tvshows")},
		{config.ConfigKeySCPUser, "test"},
		{config.ConfigKeySSHHost, "ssh.test.local"},
		{config.ConfigKeySSHKnownHosts, path.Join(os.TempDir(), ".ssh", "known_hosts")},
		{config.ConfigKeySSHPort, "22"},
		{config.ConfigKeySSHPrivateKey, path.Join(os.TempDir(), ".ssh", "id_rsa")},
		{config.ConfigKeySSHUsername, "test"},
	}

	for _, try := range tests {
		test.ExecuteCommand(t, cmd.Root, []string{"config", "set", try.key, try.value})
		test.AssertEquals(t, try.value, viper.GetString(try.key))
		content, _ := os.ReadFile(testConfigFilePath)
		subKey := strings.ReplaceAll(filepath.Ext(try.key), ".", "")
		test.AssertContains(t, fmt.Sprintf("%s=%s", subKey, try.value), string(content))
	}
}
