package info

import (
	"fmt"
	"testing"

	"github.com/jeremiergz/nas-cli/cmd"
	"github.com/jeremiergz/nas-cli/util/test"
)

func init() {
	cmd.Root.AddCommand(Cmd)
}

func TestInfoCmd(t *testing.T) {
	tests := []string{
		fmt.Sprintf("BuildDate: %s", BuildDate),
		fmt.Sprintf("Compiler:  %s", Compiler),
		fmt.Sprintf("GitCommit: %s", GitCommit),
		fmt.Sprintf("Platform:  %s", Platform),
		fmt.Sprintf("Version:   %s", Version),
	}

	_, output := test.ExecuteCommand(t, cmd.Root, []string{"info"})

	for _, try := range tests {
		test.AssertContains(t, try, output)
	}
}
