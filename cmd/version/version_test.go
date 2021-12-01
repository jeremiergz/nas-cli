package version

import (
	"fmt"
	"testing"

	"github.com/jeremiergz/nas-cli/cmd"
	"github.com/jeremiergz/nas-cli/cmd/info"
	configutil "github.com/jeremiergz/nas-cli/util/config"
	"github.com/jeremiergz/nas-cli/util/test"
)

func TestVersionCmd(t *testing.T) {
	tempDir := t.TempDir()
	configutil.Dir = tempDir

	rootCmd := cmd.NewRootCmd()
	rootCmd.AddCommand(NewVersionCmd())

	_, output := test.ExecuteCommand(t, rootCmd, []string{"version"})

	expected := fmt.Sprintf("%s %s", rootCmd.Name(), info.Version)
	test.AssertEquals(t, expected, output)
}
