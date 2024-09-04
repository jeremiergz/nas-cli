package config

import (
	"bytes"
	"fmt"
	"os/user"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jeremiergz/nas-cli/internal/cmd"
	"github.com/jeremiergz/nas-cli/internal/config"
)

func Test_Config_Get(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	output := new(bytes.Buffer)
	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	rootCMD.SetArgs([]string{"config", "get", config.KeySSHUser})

	err := rootCMD.Execute()
	assert.NoError(t, err)

	currentUser, _ := user.Current()

	assert.Equal(t, fmt.Sprintf("%s\n", currentUser.Username), output.String())
}
