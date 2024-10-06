package config

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/jeremiergz/nas-cli/internal/cmd"
	"github.com/jeremiergz/nas-cli/internal/config"
	svc "github.com/jeremiergz/nas-cli/internal/service"
)

func Test_Config_Set(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	tests := []struct {
		key   string
		value string
	}{
		{config.KeyNASFQDN, "nas.test.local"},
		{config.KeySCPChownGroup, "test"},
		{config.KeySCPChownUser, "test"},
		{config.KeySSHHost, "ssh.test.local"},
		{config.KeySSHClientKnownHosts, path.Join(os.TempDir(), ".ssh", "known_hosts")},
		{config.KeySSHPort, "22"},
		{config.KeySSHClientPrivateKey, path.Join(os.TempDir(), ".ssh", "id_rsa")},
		{config.KeySSHUser, "test"},
	}

	for _, try := range tests {
		t.Run(fmt.Sprintf("With_Key_%s", try.key), func(t *testing.T) {
			output := new(bytes.Buffer)
			rootCMD.SetOut(output)
			rootCMD.SetErr(output)
			svc.Console.SetOutput(output)
			rootCMD.SetArgs([]string{"config", "set", try.key, try.value})

			err := rootCMD.Execute()
			assert.NoError(t, err)

			assert.Equal(t, try.value, viper.GetString(try.key))
			content, _ := os.ReadFile(path.Join(tempDir, config.Filename))
			subKey := strings.ReplaceAll(filepath.Ext(try.key), ".", "")

			configLineRegExp := regexp.MustCompile(fmt.Sprintf(`%s\s*:\s*%s`, subKey, try.value))
			assert.Regexp(t, configLineRegExp, string(content))
		})
	}
}
