package config

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/jeremiergz/nas-cli/internal/cmd"
	"github.com/jeremiergz/nas-cli/internal/config"
	svc "github.com/jeremiergz/nas-cli/internal/service"
)

func Test_Config_List(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	output := new(bytes.Buffer)
	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	svc.Console.SetOutput(output)
	rootCMD.SetArgs([]string{"config", "list"})

	err := rootCMD.Execute()
	assert.NoError(t, err)

	for _, key := range viper.AllKeys() {
		t.Run(fmt.Sprintf("With_Key_%s", key), func(t *testing.T) {
			assert.Contains(t, output.String(), fmt.Sprintf("%s=", key))
		})
	}
}
