package test

import (
	"bytes"
	"os"
	"path"
	"strings"
	"testing"

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

func AssertSameTVShowTree(t *testing.T, expected map[string]map[string][]string, dir string) {
	t.Helper()
	for tvshow, seasons := range expected {
		tvshowPath := path.Join(dir, tvshow)
		dirInfo, err := os.Stat(tvshowPath)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		} else if dirInfo.IsDir() {
			for season, files := range seasons {
				seasonPath := path.Join(tvshowPath, season)
				dirInfo, err = os.Stat(seasonPath)

				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else if dirInfo.IsDir() {
					for _, file := range files {
						filePath := path.Join(seasonPath, file)
						fileInfo, err := os.Stat(filePath)

						if err != nil {
							t.Errorf("Unexpected error: %v", err)
						} else if fileInfo == nil {
							t.Fatalf("\nExpected episode to be a file: %s", filePath)
						}
					}
				} else {
					t.Fatalf("\nExpected season to be a directory: %s", seasonPath)
				}
			}
		} else {
			t.Fatalf("\nExpected TV show to be a directory: %s", tvshowPath)
		}
	}
}

func ExecuteCommand(t *testing.T, root *cobra.Command, args []string) (c *cobra.Command, output string) {
	t.Helper()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	c, err := root.ExecuteC()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	return c, strings.TrimSpace(buf.String())
}

func ExecuteCommandE(t *testing.T, root *cobra.Command, args []string) (c *cobra.Command, output string, err error) {
	t.Helper()
	c, output = ExecuteCommand(t, root, args)

	return c, output, err
}
