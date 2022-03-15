package completion

import (
	"fmt"
	"testing"

	"github.com/jeremiergz/nas-cli/cmd"
	"github.com/jeremiergz/nas-cli/config"
	"github.com/jeremiergz/nas-cli/test"
)

func TestCompletionCmd(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	rootCmd := cmd.NewRootCmd()
	rootCmd.AddCommand(NewCompletionCmd())

	tests := []struct {
		args     []string
		contains string
	}{
		{[]string{"bash"}, fmt.Sprintf("# bash completion for %s", rootCmd.Name())},
		{[]string{"zsh"}, fmt.Sprintf("# zsh completion for %s", rootCmd.Name())},
	}

	for _, try := range tests {
		args := append([]string{"completion"}, try.args...)
		_, output := test.ExecuteCommand(t, rootCmd, args)

		test.AssertContains(t, try.contains, output)
	}
}
