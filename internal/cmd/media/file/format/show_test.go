package format

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pterm/pterm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jeremiergz/nas-cli/internal/cmd"
	"github.com/jeremiergz/nas-cli/internal/config"
	svc "github.com/jeremiergz/nas-cli/internal/service"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

func Test_Show_With_Dry_Run(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	baseFiles := []string{
		"test.s01e01.mkv",
		"test.s01e02.mkv",
		"test.s01e03.mkv",
		"test.s01e04.mkv",
		"test.s01e05.mkv",
		"test.s02e01.mkv",
		"test.s02e02.mkv",
		"test.s02e03.mkv",
		"test.s02e04.mkv",
		"test.s02e05.mkv",
	}
	prepareShows(t, tempDir, baseFiles)
	lw := cmdutil.NewListWriter()
	lw.AppendItem(fmt.Sprintf("%s (1 show)", tempDir))
	lw.Indent()
	lw.AppendItem("Test (2 seasons / 10 episodes)")

	lw.Indent()
	lw.AppendItem(fmt.Sprintf("%s (%d episodes)", "Season 1", 5))
	lw.Indent()
	lw.AppendItem(fmt.Sprintf("%s  <-  %s", "Test - S01E01.mkv", pterm.Gray("test.s01e01.mkv")))
	lw.AppendItem(fmt.Sprintf("%s  <-  %s", "Test - S01E02.mkv", pterm.Gray("test.s01e02.mkv")))
	lw.AppendItem(fmt.Sprintf("%s  <-  %s", "Test - S01E03.mkv", pterm.Gray("test.s01e03.mkv")))
	lw.AppendItem(fmt.Sprintf("%s  <-  %s", "Test - S01E04.mkv", pterm.Gray("test.s01e04.mkv")))
	lw.AppendItem(fmt.Sprintf("%s  <-  %s", "Test - S01E05.mkv", pterm.Gray("test.s01e05.mkv")))
	lw.UnIndent()
	lw.UnIndent()

	lw.Indent()
	lw.AppendItem(fmt.Sprintf("%s (%d episodes)", "Season 2", 5))
	lw.Indent()
	lw.AppendItem(fmt.Sprintf("%s  <-  %s", "Test - S02E01.mkv", pterm.Gray("test.s02e01.mkv")))
	lw.AppendItem(fmt.Sprintf("%s  <-  %s", "Test - S02E02.mkv", pterm.Gray("test.s02e02.mkv")))
	lw.AppendItem(fmt.Sprintf("%s  <-  %s", "Test - S02E03.mkv", pterm.Gray("test.s02e03.mkv")))
	lw.AppendItem(fmt.Sprintf("%s  <-  %s", "Test - S02E04.mkv", pterm.Gray("test.s02e04.mkv")))
	lw.AppendItem(fmt.Sprintf("%s  <-  %s", "Test - S02E05.mkv", pterm.Gray("test.s02e05.mkv")))
	lw.UnIndent()
	lw.UnIndent()

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	args := []string{"format", "shows", "--dry-run", tempDir}

	output := new(bytes.Buffer)
	ctx := context.Background()

	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	svc.Console.SetOutput(output)
	rootCMD.SetArgs(args)
	err := rootCMD.ExecuteContext(ctx)

	assert.NoError(t, err)

	assert.Equal(t, strings.TrimSpace(lw.Render()), strings.TrimSpace(output.String()))
}

func Test_Show_Without_Options(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	rootCmd := cmd.New()
	rootCmd.AddCommand(New())

	baseFiles := []string{
		"test.s01e01.mkv",
		"test.s01e02.mkv",
		"test.s01e03.mkv",
		"test.s01e04.mkv",
		"test.s01e05.mkv",
		"test.s02e01.mkv",
		"test.s02e02.mkv",
		"test.s02e03.mkv",
		"test.s02e04.mkv",
		"test.s02e05.mkv",
	}
	expectedTree := map[string]map[string][]string{
		"Test": {
			"Season 1": {
				"Test - S01E01.mkv",
				"Test - S01E02.mkv",
				"Test - S01E03.mkv",
				"Test - S01E04.mkv",
				"Test - S01E05.mkv",
			},
			"Season 2": {
				"Test - S02E01.mkv",
				"Test - S02E02.mkv",
				"Test - S02E03.mkv",
				"Test - S02E04.mkv",
				"Test - S02E05.mkv",
			},
		},
	}
	prepareShows(t, tempDir, baseFiles)

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	args := []string{"format", "shows", "--yes", tempDir}

	output := new(bytes.Buffer)
	ctx := context.Background()

	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	svc.Console.SetOutput(output)
	rootCMD.SetArgs(args)
	err := rootCMD.ExecuteContext(ctx)

	assert.NoError(t, err)
	assertSameShowTree(t, expectedTree, tempDir)
}

func Test_Show_With_Name_Override(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	rootCmd := cmd.New()
	rootCmd.AddCommand(New())

	baseFiles := []string{
		"test.s01e01.mkv",
		"test.s01e02.mkv",
		"test.s01e03.mkv",
		"test.s01e04.mkv",
		"test.s01e05.mkv",
		"test.s02e01.mkv",
		"test.s02e02.mkv",
		"test.s02e03.mkv",
		"test.s02e04.mkv",
		"test.s02e05.mkv",
	}
	showName := "Overridden Name"
	expectedTree := map[string]map[string][]string{
		showName: {
			"Season 1": {
				fmt.Sprintf("%s - S01E01.mkv", showName),
				fmt.Sprintf("%s - S01E02.mkv", showName),
				fmt.Sprintf("%s - S01E03.mkv", showName),
				fmt.Sprintf("%s - S01E04.mkv", showName),
				fmt.Sprintf("%s - S01E05.mkv", showName),
			},
			"Season 2": {
				fmt.Sprintf("%s - S02E01.mkv", showName),
				fmt.Sprintf("%s - S02E02.mkv", showName),
				fmt.Sprintf("%s - S02E03.mkv", showName),
				fmt.Sprintf("%s - S02E04.mkv", showName),
				fmt.Sprintf("%s - S02E05.mkv", showName),
			},
		},
	}
	prepareShows(t, tempDir, baseFiles)

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	args := []string{"format", "shows", "--yes", "--name", showName, tempDir}

	output := new(bytes.Buffer)
	ctx := context.Background()

	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	svc.Console.SetOutput(output)
	rootCMD.SetArgs(args)
	err := rootCMD.ExecuteContext(ctx)

	assert.NoError(t, err)
	assertSameShowTree(t, expectedTree, tempDir)
}

func assertSameShowTree(t *testing.T, expected map[string]map[string][]string, dir string) {
	t.Helper()
	for _, seasons := range expected {
		for _, episodes := range seasons {
			for _, ep := range episodes {
				epPath := filepath.Join(dir, ep)
				require.FileExists(t, epPath)
			}
		}
	}
}

func prepareShows(t *testing.T, dir string, files []string) {
	t.Helper()
	for _, file := range files {
		os.Create(filepath.Join(dir, file))
	}
}
