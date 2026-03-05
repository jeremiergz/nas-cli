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

func Test_Movie_With_Dry_Run(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	baseFiles := []string{
		"Random.Movie.Name.1992.mkv",
	}
	prepareMovies(t, tempDir, baseFiles)

	lw := cmdutil.NewListWriter()
	lw.AppendItem(fmt.Sprintf("%s (1 movie)", tempDir))
	lw.Indent()
	lw.AppendItem(
		fmt.Sprintf(
			"%s  <-  %s",
			filepath.Join("Random Movie Name (1992).mkv", "Random Movie Name (1992).mkv.mkv"),
			pterm.Gray("Random.Movie.Name.1992.mkv"),
		),
	)

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	args := []string{"format", "movies", "--dry-run", tempDir}

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

func Test_Movie_With_Dry_Run_No_Files(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	args := []string{"format", "movies", "--dry-run", tempDir}

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

func Test_Movie_Without_Options(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	baseFiles := []string{
		"Random.Movie.Name.1992.mkv",
	}
	prepareMovies(t, tempDir, baseFiles)

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	args := []string{"format", "movies", "--yes", tempDir}

	output := new(bytes.Buffer)
	ctx := context.Background()

	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	svc.Console.SetOutput(output)
	rootCMD.SetArgs(args)
	err := rootCMD.ExecuteContext(ctx)

	assert.NoError(t, err)

	expectedPath := filepath.Join(tempDir, "Random Movie Name (1992).mkv")
	assert.FileExists(t, expectedPath)

	originalPath := filepath.Join(tempDir, "Random.Movie.Name.1992.mkv")
	_, err = os.Stat(originalPath)
	assert.True(t, os.IsNotExist(err))
}

func Test_Movie_Without_Options_Multiple(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	baseFiles := []string{
		"Random.Movie.Name.1992.mkv",
		"Another.Great.Film.2005.mkv",
		"Some.Title.2020.mkv",
	}
	prepareMovies(t, tempDir, baseFiles)

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	args := []string{"format", "movies", "--yes", tempDir}

	output := new(bytes.Buffer)
	ctx := context.Background()

	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	svc.Console.SetOutput(output)
	rootCMD.SetArgs(args)
	err := rootCMD.ExecuteContext(ctx)

	assert.NoError(t, err)

	expected := []string{
		"Random Movie Name (1992).mkv",
		"Another Great Film (2005).mkv",
		"Some Title (2020).mkv",
	}
	for _, name := range expected {
		assert.FileExists(t, filepath.Join(tempDir, name))
	}
}

func Test_Movie_ProcessMovies(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	baseFiles := []string{
		"Random.Movie.Name.1992.mkv",
	}
	prepareMovies(t, tempDir, baseFiles)

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	args := []string{"format", "movies", "--yes", tempDir}

	output := new(bytes.Buffer)
	ctx := context.Background()

	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	svc.Console.SetOutput(output)
	rootCMD.SetArgs(args)
	err := rootCMD.ExecuteContext(ctx)

	assert.NoError(t, err)

	expectedPath := filepath.Join(tempDir, "Random Movie Name (1992).mkv")
	assert.FileExists(t, expectedPath)

	info, err := os.Stat(expectedPath)
	require.NoError(t, err)
	assert.Equal(t, config.FileMode, info.Mode().Perm())
}

func prepareMovies(t *testing.T, dir string, files []string) {
	t.Helper()
	for _, file := range files {
		os.Create(filepath.Join(dir, file))
	}
}
