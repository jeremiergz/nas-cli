package version

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jeremiergz/nas-cli/internal/cmd"
	"github.com/jeremiergz/nas-cli/internal/config"
)

func Test_Outputs_The_Correct_Version(t *testing.T) {
	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	tests := []string{
		"N/A",
		"v1.0.0",
		"qys2toaiqdignhk88z39o3hw5zm3234wv9qfx6uz",
	}

	for _, test := range tests {
		config.Version = test
		output := new(bytes.Buffer)
		rootCMD.SetOut(output)
		rootCMD.SetErr(output)
		rootCMD.SetArgs([]string{"version"})
		rootCMD.Execute()

		expected := fmt.Sprintf("%s %s\n", rootCMD.Name(), test)

		assert.Equal(t, expected, output.String())
	}
}
