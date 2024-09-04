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
	mediasvc "github.com/jeremiergz/nas-cli/internal/service/media"
	mkvsvc "github.com/jeremiergz/nas-cli/internal/service/mkv"
	sftpsvc "github.com/jeremiergz/nas-cli/internal/service/sftp"
	sshsvc "github.com/jeremiergz/nas-cli/internal/service/ssh"
	"github.com/jeremiergz/nas-cli/internal/util/ctxutil"
)

func Test_TV_Show_With_Dry_Run(t *testing.T) {
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
	prepareTVShows(t, tempDir, baseFiles)
	rootTree := gotree.New(tempDir)
	tvShowTree := rootTree.Add("Test")

	season1Tree := tvShowTree.Add(fmt.Sprintf("%s (%d files)", "Season 1", 5))
	season1Tree.Add(fmt.Sprintf("%s  %s", "Test - S01E01.mkv", "test.s01e01.mkv"))
	season1Tree.Add(fmt.Sprintf("%s  %s", "Test - S01E02.mkv", "test.s01e02.mkv"))
	season1Tree.Add(fmt.Sprintf("%s  %s", "Test - S01E03.mkv", "test.s01e03.mkv"))
	season1Tree.Add(fmt.Sprintf("%s  %s", "Test - S01E04.mkv", "test.s01e04.mkv"))
	season1Tree.Add(fmt.Sprintf("%s  %s", "Test - S01E05.mkv", "test.s01e05.mkv"))

	season2Tree := tvShowTree.Add(fmt.Sprintf("%s (%d files)", "Season 2", 5))
	season2Tree.Add(fmt.Sprintf("%s  %s", "Test - S02E01.mkv", "test.s02e01.mkv"))
	season2Tree.Add(fmt.Sprintf("%s  %s", "Test - S02E02.mkv", "test.s02e02.mkv"))
	season2Tree.Add(fmt.Sprintf("%s  %s", "Test - S02E03.mkv", "test.s02e03.mkv"))
	season2Tree.Add(fmt.Sprintf("%s  %s", "Test - S02E04.mkv", "test.s02e04.mkv"))
	season2Tree.Add(fmt.Sprintf("%s  %s", "Test - S02E05.mkv", "test.s02e05.mkv"))

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	args := []string{"format", "tvshows", "--dry-run", tempDir}

	output := new(bytes.Buffer)
	ctx := getTestContext(t, output)

	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	rootCMD.SetArgs(args)
	err := rootCMD.ExecuteContext(ctx)

	assert.NoError(t, err)

	assert.Equal(t, strings.TrimSpace(rootTree.Print()), strings.TrimSpace(output.String()))
}

func Test_TV_Show_Without_Options(t *testing.T) {
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
	prepareTVShows(t, tempDir, baseFiles)

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	args := []string{"format", "tvshows", "--yes", tempDir}

	output := new(bytes.Buffer)
	ctx := getTestContext(t, output)

	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	rootCMD.SetArgs(args)
	err := rootCMD.ExecuteContext(ctx)

	assert.NoError(t, err)
	assertSameTVShowTree(t, expectedTree, tempDir)
}

func Test_TV_Show_With_Name_Override(t *testing.T) {
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
	tvshowName := "Overridden Name"
	expectedTree := map[string]map[string][]string{
		tvshowName: {
			"Season 1": {
				fmt.Sprintf("%s - S01E01.mkv", tvshowName),
				fmt.Sprintf("%s - S01E02.mkv", tvshowName),
				fmt.Sprintf("%s - S01E03.mkv", tvshowName),
				fmt.Sprintf("%s - S01E04.mkv", tvshowName),
				fmt.Sprintf("%s - S01E05.mkv", tvshowName),
			},
			"Season 2": {
				fmt.Sprintf("%s - S02E01.mkv", tvshowName),
				fmt.Sprintf("%s - S02E02.mkv", tvshowName),
				fmt.Sprintf("%s - S02E03.mkv", tvshowName),
				fmt.Sprintf("%s - S02E04.mkv", tvshowName),
				fmt.Sprintf("%s - S02E05.mkv", tvshowName),
			},
		},
	}
	prepareTVShows(t, tempDir, baseFiles)

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	args := []string{"format", "tvshows", "--yes", "--name", tvshowName, tempDir}

	output := new(bytes.Buffer)
	ctx := getTestContext(t, output)

	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	rootCMD.SetArgs(args)
	err := rootCMD.ExecuteContext(ctx)

	assert.NoError(t, err)
	assertSameTVShowTree(t, expectedTree, tempDir)
}

func assertSameTVShowTree(t *testing.T, expected map[string]map[string][]string, dir string) {
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

func getTestContext(t *testing.T, w io.Writer) context.Context {
	t.Helper()

	ctx := context.Background()
	ctx = ctxutil.WithSingleton(ctx, consolesvc.New(w))
	ctx = ctxutil.WithSingleton(ctx, mediasvc.New())
	ctx = ctxutil.WithSingleton(ctx, mkvsvc.New())
	ctx = ctxutil.WithSingleton(ctx, sftpsvc.New())
	ctx = ctxutil.WithSingleton(ctx, sshsvc.New())

	return ctx
}

func prepareTVShows(t *testing.T, dir string, files []string) {
	t.Helper()
	for _, file := range files {
		os.Create(path.Join(dir, file))
	}
}
