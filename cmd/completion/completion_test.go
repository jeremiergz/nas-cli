package completion

import (
	"fmt"
	"testing"

	"github.com/jeremiergz/nas-cli/cmd"
	"github.com/jeremiergz/nas-cli/util/test"
)

func init() {
	cmd.Root.AddCommand(Cmd)
}

func TestCompletionCmd(t *testing.T) {
	tests := []struct {
		args     []string
		contains string
	}{
		{[]string{"bash"}, fmt.Sprintf("# bash completion for %s", cmd.Root.Name())},
		{[]string{"zsh"}, fmt.Sprintf("# zsh completion for %s", cmd.Root.Name())},
	}

	for _, try := range tests {
		args := append([]string{"completion"}, try.args...)
		_, output := test.ExecuteCommand(t, cmd.Root, args)

		test.AssertContains(t, try.contains, output)
	}
}
