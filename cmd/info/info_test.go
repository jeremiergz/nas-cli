package info

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jeremiergz/nas-cli/cmd"
	"github.com/jeremiergz/nas-cli/util/processutil"
)

func Test_Outputs_Information_As_JSON(t *testing.T) {
	output, err := testInfoOutput(t, "json")

	assert.NoError(t, err)
	expected := fmt.Sprintf(`{
  "buildDate": "%s",
  "compiler": "%s",
  "gitCommit": "%s",
  "platform": "%s",
  "version": "%s"
}`+"\n",
		processutil.BuildDate, processutil.Compiler, processutil.GitCommit, processutil.Platform, processutil.Version,
	)
	assert.Equal(t, expected, output)
}

func Test_Outputs_Information_As_TEXT(t *testing.T) {
	output, err := testInfoOutput(t, "text")

	assert.NoError(t, err)
	expected := fmt.Sprintf(`BuildDate: %s
Compiler:  %s
GitCommit: %s
Platform:  %s
Version:   %s
`, processutil.BuildDate, processutil.Compiler, processutil.GitCommit, processutil.Platform, processutil.Version,
	)
	assert.Equal(t, expected, output)
}

func Test_Outputs_Information_As_YAML(t *testing.T) {
	output, err := testInfoOutput(t, "yaml")

	assert.NoError(t, err)
	expected := fmt.Sprintf(
		"buildDate: \"%s\"\ncompiler: %s\ngitCommit: %s\nplatform: %s\nversion: %s\n",
		processutil.BuildDate, processutil.Compiler, processutil.GitCommit, processutil.Platform, processutil.Version,
	)
	assert.Equal(t, expected, output)
}

func Test_Fails_When(t *testing.T) {
	testInfoOutput(t, "unknown")

	output, err := testInfoOutput(t, "unknown")

	assert.Error(t, err)
	assert.Contains(t, output, "invalid value unknown")
}

func testInfoOutput(t *testing.T, format string) (string, error) {
	t.Helper()

	rootCMD := cmd.NewCommand()
	rootCMD.AddCommand(NewCommand())

	output := new(bytes.Buffer)
	rootCMD.SetOut(output)
	rootCMD.SetErr(output)

	processutil.BuildDate = "1970-01-01T00:00:00.000Z"
	processutil.Compiler = "gc/test"
	processutil.GitCommit = "pwzeau3eyn9cb5qnxyb657ihuj6iymyefd8rs53m"
	processutil.Platform = "test/arm64"
	processutil.Version = "1970.01.01"

	args := []string{"info", "--output=" + format}
	rootCMD.SetArgs(args)
	err := rootCMD.Execute()

	return output.String(), err
}
