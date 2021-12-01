package config

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/cmd"
	configutil "github.com/jeremiergz/nas-cli/util/config"
	"github.com/jeremiergz/nas-cli/util/test"
)

func TestConfigGetCmd(t *testing.T) {
	tempDir := t.TempDir()
	configutil.Dir = tempDir

	rootCmd := cmd.NewRootCmd()
	rootCmd.AddCommand(NewConfigCmd())

	_, output := test.ExecuteCommand(t, rootCmd, []string{"config", "get", configutil.ConfigKeySSHUsername})
	currentUser, _ := user.Current()

	content, _ := os.ReadFile(path.Join(tempDir, configutil.FileName))
	fmt.Println(string(content))

	test.AssertEquals(t, currentUser.Username, output)
}

func TestConfigListCmd(t *testing.T) {
	tempDir := t.TempDir()
	configutil.Dir = tempDir

	rootCmd := cmd.NewRootCmd()
	rootCmd.AddCommand(NewConfigCmd())

	_, output := test.ExecuteCommand(t, rootCmd, []string{"config", "list"})

	for _, key := range viper.AllKeys() {
		test.AssertContains(t, fmt.Sprintf("%s=", key), output)
	}
}

func TestConfigSetCmd(t *testing.T) {
	tempDir := t.TempDir()
	configutil.Dir = tempDir

	rootCmd := cmd.NewRootCmd()
	rootCmd.AddCommand(NewConfigCmd())

	tests := []struct {
		key   string
		value string
	}{
		{configutil.ConfigKeyNASDomain, "nas.test.local"},
		{configutil.ConfigKeySCPAnimes, path.Join(os.TempDir(), "animes")},
		{configutil.ConfigKeySCPGroup, "test"},
		{configutil.ConfigKeySCPMovies, path.Join(os.TempDir(), "movies")},
		{configutil.ConfigKeySCPTVShows, path.Join(os.TempDir(), "tvshows")},
		{configutil.ConfigKeySCPUser, "test"},
		{configutil.ConfigKeySSHHost, "ssh.test.local"},
		{configutil.ConfigKeySSHKnownHosts, path.Join(os.TempDir(), ".ssh", "known_hosts")},
		{configutil.ConfigKeySSHPort, "22"},
		{configutil.ConfigKeySSHPrivateKey, path.Join(os.TempDir(), ".ssh", "id_rsa")},
		{configutil.ConfigKeySSHUsername, "test"},
	}

	for _, try := range tests {
		test.ExecuteCommand(t, rootCmd, []string{"config", "set", try.key, try.value})
		test.AssertEquals(t, try.value, viper.GetString(try.key))
		content, _ := os.ReadFile(path.Join(tempDir, configutil.FileName))
		subKey := strings.ReplaceAll(filepath.Ext(try.key), ".", "")
		test.AssertContains(t, fmt.Sprintf("%s=%s", subKey, try.value), string(content))
	}
}
