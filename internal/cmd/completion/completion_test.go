package completion

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jeremiergz/nas-cli/internal/cmd"
)

func Test_Completion_For_Bash(t *testing.T) {
	output, err := testCompletionGeneration(t, "bash")

	assert.NoError(t, err)
	assert.Contains(t, output, "# bash completion for")
}

func Test_Completion_For_ZSH(t *testing.T) {
	output, err := testCompletionGeneration(t, "zsh")

	assert.NoError(t, err)
	assert.Contains(t, output, "# zsh completion for")
}

func Test_Completion_Fails_When_Shell_Is_Unknown(t *testing.T) {
	output, err := testCompletionGeneration(t, "unknown")

	assert.Contains(t, output, "invalid argument \"unknown\"")
	assert.Error(t, err)
}

func testCompletionGeneration(t *testing.T, shell string) (string, error) {
	t.Helper()

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	args := []string{"completion", shell}

	output := new(bytes.Buffer)
	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	rootCMD.SetArgs(args)
	err := rootCMD.Execute()

	return output.String(), err
}
