package format

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/disiqueira/gotree/v3"

	"github.com/jeremiergz/nas-cli/cmd"
	configutil "github.com/jeremiergz/nas-cli/util/config"
	"github.com/jeremiergz/nas-cli/util/test"
)

func prepareTVShows(t *testing.T, dir string, files []string) {
	t.Helper()
	for _, file := range files {
		os.Create(path.Join(dir, file))
	}
}

func TestTVShowCmdWithDryRun(t *testing.T) {
	tempDir := t.TempDir()
	configutil.Dir = tempDir

	rootCmd := cmd.NewRootCmd()
	rootCmd.AddCommand(NewFormatCmd())

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

	_, output, _ := test.ExecuteCommandE(t, rootCmd, []string{"format", "tvshows", "--dry-run", tempDir})
	fmt.Println(output)
	fmt.Println(os.ReadDir(tempDir))
	test.AssertEquals(t, strings.TrimSpace(rootTree.Print()), strings.TrimSpace(output))
}

func TestTVShowCmdWithoutOptions(t *testing.T) {
	tempDir := t.TempDir()
	configutil.Dir = tempDir

	rootCmd := cmd.NewRootCmd()
	rootCmd.AddCommand(NewFormatCmd())

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
	test.ExecuteCommand(t, rootCmd, []string{"format", "tvshows", "--yes", tempDir})
	test.AssertSameTVShowTree(t, expectedTree, tempDir)
}

func TestTVShowCmdWithNameOverride(t *testing.T) {
	tempDir := t.TempDir()
	configutil.Dir = tempDir

	rootCmd := cmd.NewRootCmd()
	rootCmd.AddCommand(NewFormatCmd())

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
	tvshowName := "Overriden Name"
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
	test.ExecuteCommand(t, rootCmd, []string{"format", "tvshows", "--yes", "--name", tvshowName, tempDir})
	test.AssertSameTVShowTree(t, expectedTree, tempDir)
}
