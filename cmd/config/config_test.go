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
	"github.com/jeremiergz/nas-cli/config"
	"github.com/jeremiergz/nas-cli/test"
)

func TestConfigGetCmd(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	rootCmd := cmd.NewRootCmd()
	rootCmd.AddCommand(NewConfigCmd())

	_, output := test.ExecuteCommand(t, rootCmd, []string{"config", "get", config.KeySSHUser})
	currentUser, _ := user.Current()

	content, _ := os.ReadFile(path.Join(tempDir, config.FileName))
	fmt.Println(string(content))

	test.AssertEquals(t, currentUser.Username, output)
}

func TestConfigListCmd(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	rootCmd := cmd.NewRootCmd()
	rootCmd.AddCommand(NewConfigCmd())

	_, output := test.ExecuteCommand(t, rootCmd, []string{"config", "list"})

	for _, key := range viper.AllKeys() {
		test.AssertContains(t, fmt.Sprintf("%s=", key), output)
	}
}

func TestConfigSetCmd(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	rootCmd := cmd.NewRootCmd()
	rootCmd.AddCommand(NewConfigCmd())

	tests := []struct {
		key   string
		value string
	}{
		{config.KeyNASFQDN, "nas.test.local"},
		{config.KeySCPDestAnimesPath, path.Join(os.TempDir(), "animes")},
		{config.KeySCPChownGroup, "test"},
		{config.KeySCPDestMoviesPath, path.Join(os.TempDir(), "movies")},
		{config.KeySCPDestTVShowsPath, path.Join(os.TempDir(), "tvshows")},
		{config.KeySCPChownUser, "test"},
		{config.KeySSHHost, "ssh.test.local"},
		{config.KeySSHClientKnownHosts, path.Join(os.TempDir(), ".ssh", "known_hosts")},
		{config.KeySSHPort, "22"},
		{config.KeySSHClientPrivateKey, path.Join(os.TempDir(), ".ssh", "id_rsa")},
		{config.KeySSHUser, "test"},
	}

	for _, try := range tests {
		test.ExecuteCommand(t, rootCmd, []string{"config", "set", try.key, try.value})
		test.AssertEquals(t, try.value, viper.GetString(try.key))
		content, _ := os.ReadFile(path.Join(tempDir, config.FileName))
		subKey := strings.ReplaceAll(filepath.Ext(try.key), ".", "")
		test.AssertContains(t, fmt.Sprintf("%s=%s", subKey, try.value), string(content))
	}
}
