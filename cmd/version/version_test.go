package version

import (
	"fmt"
	"testing"

	"github.com/jeremiergz/nas-cli/cmd"
	"github.com/jeremiergz/nas-cli/cmd/info"
	"github.com/jeremiergz/nas-cli/config"
	"github.com/jeremiergz/nas-cli/test"
)

func TestVersionCmd(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	rootCmd := cmd.NewRootCmd()
	rootCmd.AddCommand(NewVersionCmd())

	_, output := test.ExecuteCommand(t, rootCmd, []string{"version"})

	expected := fmt.Sprintf("%s %s", rootCmd.Name(), info.Version)
	test.AssertEquals(t, expected, output)
}
