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
	"github.com/jeremiergz/nas-cli/internal/model"
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

func Test_Show_With_Dry_Run_No_Files(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

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
	assert.Contains(t, output.String(), "Nothing to process")
}

func Test_Show_With_Yes_No_Files(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

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
	assert.Contains(t, output.String(), "Nothing to process")
}

func Test_Show_With_Invalid_Directory(t *testing.T) {
	config.Dir = t.TempDir()

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	args := []string{"format", "shows", "--yes", "/nonexistent/path"}

	output := new(bytes.Buffer)
	ctx := context.Background()

	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	svc.Console.SetOutput(output)
	rootCMD.SetArgs(args)
	err := rootCMD.ExecuteContext(ctx)

	assert.Error(t, err)
}

func Test_Show_With_Name_Count_Mismatch(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	baseFiles := []string{
		"test.s01e01.mkv",
	}
	prepareShows(t, tempDir, baseFiles)

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	// Provide 2 names for 1 show.
	args := []string{"format", "shows", "--yes", "--name", "Name1", "--name", "Name2", tempDir}

	output := new(bytes.Buffer)
	ctx := context.Background()

	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	svc.Console.SetOutput(output)
	rootCMD.SetArgs(args)
	err := rootCMD.ExecuteContext(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "names must be provided for all shows")
}

func Test_Show_With_Extension_Filter(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	prepareShows(t, tempDir, []string{
		"test.s01e01.mkv",
		"test.s01e02.avi",
	})

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	// Only process mkv files.
	args := []string{"format", "shows", "--yes", "--ext", "mkv", tempDir}

	output := new(bytes.Buffer)
	ctx := context.Background()

	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	svc.Console.SetOutput(output)
	rootCMD.SetArgs(args)
	err := rootCMD.ExecuteContext(ctx)

	assert.NoError(t, err)

	// mkv file should be renamed.
	assert.FileExists(t, filepath.Join(tempDir, "Test - S01E01.mkv"))
	// avi file should remain unchanged.
	assert.FileExists(t, filepath.Join(tempDir, "test.s01e02.avi"))
}

func prepareShows(t *testing.T, dir string, files []string) {
	t.Helper()
	for _, file := range files {
		os.Create(filepath.Join(dir, file))
	}
}

func Test_Show_ProcessShows_With_Confirm_Accept(t *testing.T) {
	tempDir := t.TempDir()
	prepareShows(t, tempDir, []string{
		"test.s01e01.mkv",
		"test.s01e02.mkv",
	})

	shows, err := model.Shows(tempDir, []string{"mkv"}, false, "", nil, false)
	require.NoError(t, err)
	require.Len(t, shows, 1)

	// Confirm show, confirm season.
	p := &mockPrompter{
		confirmResults: []mockConfirmResult{
			{confirmed: true},
			{confirmed: true},
		},
	}

	output := new(bytes.Buffer)
	svc.Console.SetOutput(output)

	err = processShows(context.Background(), output, tempDir, shows, os.Getuid(), os.Getgid(), p)
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(tempDir, "Test - S01E01.mkv"))
	assert.FileExists(t, filepath.Join(tempDir, "Test - S01E02.mkv"))
}

func Test_Show_ProcessShows_With_Show_Decline(t *testing.T) {
	tempDir := t.TempDir()
	prepareShows(t, tempDir, []string{
		"test.s01e01.mkv",
	})

	shows, err := model.Shows(tempDir, []string{"mkv"}, false, "", nil, false)
	require.NoError(t, err)

	// Decline the show.
	p := &mockPrompter{
		confirmResults: []mockConfirmResult{
			{confirmed: false},
		},
	}

	output := new(bytes.Buffer)
	err = processShows(context.Background(), output, tempDir, shows, os.Getuid(), os.Getgid(), p)
	assert.NoError(t, err)

	// File should NOT be renamed.
	assert.FileExists(t, filepath.Join(tempDir, "test.s01e01.mkv"))
}

func Test_Show_ProcessShows_With_Show_Interrupt(t *testing.T) {
	tempDir := t.TempDir()
	prepareShows(t, tempDir, []string{
		"test.s01e01.mkv",
	})

	shows, err := model.Shows(tempDir, []string{"mkv"}, false, "", nil, false)
	require.NoError(t, err)

	p := &mockPrompter{
		confirmResults: []mockConfirmResult{
			{confirmed: false, err: fmt.Errorf("interrupted")},
		},
	}

	output := new(bytes.Buffer)
	err = processShows(context.Background(), output, tempDir, shows, os.Getuid(), os.Getgid(), p)
	assert.NoError(t, err)

	// File should NOT be renamed.
	assert.FileExists(t, filepath.Join(tempDir, "test.s01e01.mkv"))
}

func Test_Show_ProcessShows_With_Season_Decline(t *testing.T) {
	tempDir := t.TempDir()
	prepareShows(t, tempDir, []string{
		"test.s01e01.mkv",
		"test.s02e01.mkv",
	})

	shows, err := model.Shows(tempDir, []string{"mkv"}, false, "", nil, false)
	require.NoError(t, err)

	// Confirm show, decline season 1, confirm season 2.
	p := &mockPrompter{
		confirmResults: []mockConfirmResult{
			{confirmed: true},
			{confirmed: false},
			{confirmed: true},
		},
	}

	output := new(bytes.Buffer)
	svc.Console.SetOutput(output)

	err = processShows(context.Background(), output, tempDir, shows, os.Getuid(), os.Getgid(), p)
	assert.NoError(t, err)

	// Season 1 should NOT be renamed.
	assert.FileExists(t, filepath.Join(tempDir, "test.s01e01.mkv"))
	// Season 2 should be renamed.
	assert.FileExists(t, filepath.Join(tempDir, "Test - S02E01.mkv"))
}

func Test_Show_ProcessShows_With_Season_Interrupt(t *testing.T) {
	tempDir := t.TempDir()
	prepareShows(t, tempDir, []string{
		"test.s01e01.mkv",
	})

	shows, err := model.Shows(tempDir, []string{"mkv"}, false, "", nil, false)
	require.NoError(t, err)

	// Confirm show, interrupt at season.
	p := &mockPrompter{
		confirmResults: []mockConfirmResult{
			{confirmed: true},
			{confirmed: false, err: fmt.Errorf("interrupted")},
		},
	}

	output := new(bytes.Buffer)
	err = processShows(context.Background(), output, tempDir, shows, os.Getuid(), os.Getgid(), p)
	assert.NoError(t, err)

	// File should NOT be renamed.
	assert.FileExists(t, filepath.Join(tempDir, "test.s01e01.mkv"))
}

func Test_Show_ProcessShows_Rename_Error(t *testing.T) {
	tempDir := t.TempDir()
	prepareShows(t, tempDir, []string{
		"test.s01e01.mkv",
	})

	shows, err := model.Shows(tempDir, []string{"mkv"}, false, "", nil, false)
	require.NoError(t, err)

	// Remove the file after parsing to trigger rename error.
	os.Remove(filepath.Join(tempDir, "test.s01e01.mkv"))

	p := &mockPrompter{
		confirmResults: []mockConfirmResult{
			{confirmed: true},
			{confirmed: true},
		},
	}

	output := new(bytes.Buffer)
	err = processShows(context.Background(), output, tempDir, shows, os.Getuid(), os.Getgid(), p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not rename")
}
