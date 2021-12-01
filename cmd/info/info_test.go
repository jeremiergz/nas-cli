package info

import (
	"fmt"
	"testing"

	"github.com/jeremiergz/nas-cli/cmd"
	configutil "github.com/jeremiergz/nas-cli/util/config"
	"github.com/jeremiergz/nas-cli/util/test"
)

func TestInfoCmd(t *testing.T) {
	tempDir := t.TempDir()
	configutil.Dir = tempDir

	rootCmd := cmd.NewRootCmd()
	rootCmd.AddCommand(NewInfoCmd())

	tests := []string{
		fmt.Sprintf("BuildDate: %s", BuildDate),
		fmt.Sprintf("Compiler:  %s", Compiler),
		fmt.Sprintf("GitCommit: %s", GitCommit),
		fmt.Sprintf("Platform:  %s", Platform),
		fmt.Sprintf("Version:   %s", Version),
	}

	_, output := test.ExecuteCommand(t, rootCmd, []string{"info"})

	for _, try := range tests {
		test.AssertContains(t, try, output)
	}
}
