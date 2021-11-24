package test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jeremiergz/nas-cli/cmd"
	"github.com/spf13/cobra"
)

func AssertEquals(t *testing.T, expected, output string) {
	t.Helper()
	if output != expected {
		t.Fatalf("\nExpected: \"%v\"\nGot:      \"%v\"", expected, output)
	}
}

func AssertContains(t *testing.T, expected, output string) {
	t.Helper()
	if !strings.Contains(output, expected) {
		t.Fatalf("\nExpected to contain: \"%v\"", expected)
	}
}

func ExecuteCommand(t *testing.T, root *cobra.Command, args []string) (c *cobra.Command, output string) {
	t.Helper()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	c, err := cmd.Root.ExecuteC()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	return c, strings.TrimSpace(buf.String())
}
