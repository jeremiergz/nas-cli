package completion

import (
	"fmt"
	"testing"

	"github.com/jeremiergz/nas-cli/cmd"
	configutil "github.com/jeremiergz/nas-cli/util/config"
	"github.com/jeremiergz/nas-cli/util/test"
)

func TestCompletionCmd(t *testing.T) {
	tempDir := t.TempDir()
	configutil.Dir = tempDir

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
