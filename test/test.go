package test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/service"
	"github.com/jeremiergz/nas-cli/util"
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

func AssertContainsRegExp(t *testing.T, expected *regexp.Regexp, output string) {
	t.Helper()
	if !expected.MatchString(output) {
		t.Fatalf("\nExpected to contain RegExp: \"%v\"", expected)
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

	reader, writer, _ := os.Pipe()
	defer reader.Close()

	os.Stdout = writer
	root.SetOut(writer)
	os.Stderr = writer
	root.SetErr(writer)

	root.SetArgs(args)

	ctx := GetTestContext()
	c, err := root.ExecuteContextC(ctx)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	out := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, reader)
		out <- buf.String()
	}()

	writer.Close()

	output = <-out

	return c, strings.TrimSpace(output)
}

func ExecuteCommandE(t *testing.T, root *cobra.Command, args []string) (c *cobra.Command, output string, err error) {
	t.Helper()
	c, output = ExecuteCommand(t, root, args)

	return c, output, err
}

func GetTestContext() context.Context {
	ctx := context.Background()

	console := service.NewConsoleService(ctx)
	media := service.NewMediaService(ctx)
	sftp := service.NewSFTPService(ctx)
	ssh := service.NewSSHService(ctx)

	ctx = context.WithValue(ctx, util.ContextKeyConsole, console)
	ctx = context.WithValue(ctx, util.ContextKeyMedia, media)
	ctx = context.WithValue(ctx, util.ContextKeySFTP, sftp)
	ctx = context.WithValue(ctx, util.ContextKeySSH, ssh)

	return ctx
}
