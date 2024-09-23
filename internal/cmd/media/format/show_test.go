package format

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/disiqueira/gotree/v3"
	"github.com/stretchr/testify/assert"

	"github.com/jeremiergz/nas-cli/internal/cmd"
	"github.com/jeremiergz/nas-cli/internal/config"
	consolesvc "github.com/jeremiergz/nas-cli/internal/service/console"
	sftpsvc "github.com/jeremiergz/nas-cli/internal/service/sftp"
	sshsvc "github.com/jeremiergz/nas-cli/internal/service/ssh"
	"github.com/jeremiergz/nas-cli/internal/util/ctxutil"
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
	rootTree := gotree.New(tempDir)
	showTree := rootTree.Add("Test (2 seasons - 10 episodes)")

	season1Tree := showTree.Add(fmt.Sprintf("%s (%d episodes)", "Season 1", 5))
	season1Tree.Add(fmt.Sprintf("%s  %s", "Test - S01E01.mkv", "test.s01e01.mkv"))
	season1Tree.Add(fmt.Sprintf("%s  %s", "Test - S01E02.mkv", "test.s01e02.mkv"))
	season1Tree.Add(fmt.Sprintf("%s  %s", "Test - S01E03.mkv", "test.s01e03.mkv"))
	season1Tree.Add(fmt.Sprintf("%s  %s", "Test - S01E04.mkv", "test.s01e04.mkv"))
	season1Tree.Add(fmt.Sprintf("%s  %s", "Test - S01E05.mkv", "test.s01e05.mkv"))

	season2Tree := showTree.Add(fmt.Sprintf("%s (%d episodes)", "Season 2", 5))
	season2Tree.Add(fmt.Sprintf("%s  %s", "Test - S02E01.mkv", "test.s02e01.mkv"))
	season2Tree.Add(fmt.Sprintf("%s  %s", "Test - S02E02.mkv", "test.s02e02.mkv"))
	season2Tree.Add(fmt.Sprintf("%s  %s", "Test - S02E03.mkv", "test.s02e03.mkv"))
	season2Tree.Add(fmt.Sprintf("%s  %s", "Test - S02E04.mkv", "test.s02e04.mkv"))
	season2Tree.Add(fmt.Sprintf("%s  %s", "Test - S02E05.mkv", "test.s02e05.mkv"))

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	args := []string{"format", "shows", "--dry-run", tempDir}

	output := new(bytes.Buffer)
	ctx := getTestContext(t, output)

	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	rootCMD.SetArgs(args)
	err := rootCMD.ExecuteContext(ctx)

	assert.NoError(t, err)

	assert.Equal(t, strings.TrimSpace(rootTree.Print()), strings.TrimSpace(output.String()))
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
	ctx := getTestContext(t, output)

	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	rootCMD.SetArgs(args)
	err := rootCMD.ExecuteContext(ctx)

	assert.NoError(t, err)
	assertSameshowTree(t, expectedTree, tempDir)
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
	ctx := getTestContext(t, output)

	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	rootCMD.SetArgs(args)
	err := rootCMD.ExecuteContext(ctx)

	assert.NoError(t, err)
	assertSameshowTree(t, expectedTree, tempDir)
}

func assertSameshowTree(t *testing.T, expected map[string]map[string][]string, dir string) {
	t.Helper()
	for show, seasons := range expected {
		showPath := path.Join(dir, show)
		dirInfo, err := os.Stat(showPath)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		} else if dirInfo.IsDir() {
			for season, files := range seasons {
				seasonPath := path.Join(showPath, season)
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
			t.Fatalf("\nExpected show to be a directory: %s", showPath)
		}
	}
}

func getTestContext(t *testing.T, w io.Writer) context.Context {
	t.Helper()

	ctx := context.Background()
	ctx = ctxutil.WithSingleton(ctx, consolesvc.New(w))
	ctx = ctxutil.WithSingleton(ctx, sftpsvc.New())
	ctx = ctxutil.WithSingleton(ctx, sshsvc.New())

	return ctx
}

func prepareShows(t *testing.T, dir string, files []string) {
	t.Helper()
	for _, file := range files {
		os.Create(path.Join(dir, file))
	}
}
