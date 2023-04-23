package config

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/jeremiergz/nas-cli/cmd"
	"github.com/jeremiergz/nas-cli/config"
)

func Test_Config_List(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	rootCMD := cmd.NewCommand()
	rootCMD.AddCommand(NewCommand())

	output := new(bytes.Buffer)
	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	rootCMD.SetArgs([]string{"config", "list"})

	err := rootCMD.Execute()
	assert.NoError(t, err)

	for _, key := range viper.AllKeys() {
		key := key
		t.Run(fmt.Sprintf("With_Key_%s", key), func(t *testing.T) {
			assert.Contains(t, output.String(), fmt.Sprintf("%s=", key))
		})
	}
}
