package version

import (
	"fmt"
	"testing"

	"github.com/jeremiergz/nas-cli/cmd"
	"github.com/jeremiergz/nas-cli/cmd/info"
	"github.com/jeremiergz/nas-cli/util/test"
)

func init() {
	cmd.Root.AddCommand(Cmd)
}

func TestVersionCmd(t *testing.T) {
	_, output := test.ExecuteCommand(t, cmd.Root, []string{"version"})

	expected := fmt.Sprintf("%s %s", cmd.Root.Name(), info.Version)
	test.AssertEquals(t, expected, output)
}
