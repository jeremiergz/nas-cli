package info

import (
	"fmt"
	"testing"

	"github.com/jeremiergz/nas-cli/cmd"
	"github.com/jeremiergz/nas-cli/config"
	"github.com/jeremiergz/nas-cli/test"
)

func TestInfoCmd(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

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
