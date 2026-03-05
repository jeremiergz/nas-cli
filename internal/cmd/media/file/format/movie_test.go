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

// Records calls and returns pre-configured responses.
type mockPrompter struct {
	confirmResults []mockConfirmResult
	inputResults   []mockInputResult
	confirmIndex   int
	inputIndex     int
}

type mockConfirmResult struct {
	confirmed bool
	err       error
}

type mockInputResult struct {
	value string
	err   error
}

func (m *mockPrompter) Confirm(_ string) (bool, error) {
	if m.confirmIndex >= len(m.confirmResults) {
		return true, nil
	}
	r := m.confirmResults[m.confirmIndex]
	m.confirmIndex++
	return r.confirmed, r.err
}

func (m *mockPrompter) Input(_, defaultValue string) (string, error) {
	if m.inputIndex >= len(m.inputResults) {
		return defaultValue, nil
	}
	r := m.inputResults[m.inputIndex]
	m.inputIndex++
	return r.value, r.err
}

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

func Test_Movie_With_Yes_No_Files(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

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
	assert.Contains(t, output.String(), "Nothing to process")
}

func Test_Movie_With_Invalid_Directory(t *testing.T) {
	config.Dir = t.TempDir()

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	args := []string{"format", "movies", "--yes", "/nonexistent/path"}

	output := new(bytes.Buffer)
	ctx := context.Background()

	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	svc.Console.SetOutput(output)
	rootCMD.SetArgs(args)
	err := rootCMD.ExecuteContext(ctx)

	assert.Error(t, err)
}

func Test_Movie_With_Extension_Filter(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	// Create both mkv and avi files.
	prepareMovies(t, tempDir, []string{
		"Random.Movie.Name.1992.mkv",
		"Another.Movie.2000.avi",
	})

	rootCMD := cmd.New()
	rootCMD.AddCommand(New())

	// Only process mkv files.
	args := []string{"format", "movies", "--yes", "--ext", "mkv", tempDir}

	output := new(bytes.Buffer)
	ctx := context.Background()

	rootCMD.SetOut(output)
	rootCMD.SetErr(output)
	svc.Console.SetOutput(output)
	rootCMD.SetArgs(args)
	err := rootCMD.ExecuteContext(ctx)

	assert.NoError(t, err)

	// mkv file should be renamed.
	assert.FileExists(t, filepath.Join(tempDir, "Random Movie Name (1992).mkv"))
	// avi file should remain unchanged.
	assert.FileExists(t, filepath.Join(tempDir, "Another.Movie.2000.avi"))
}

func Test_Movie_Rename_Error(t *testing.T) {
	tempDir := t.TempDir()
	config.Dir = tempDir

	baseFiles := []string{
		"Random.Movie.Name.1992.mkv",
	}
	prepareMovies(t, tempDir, baseFiles)

	// Remove the source file to trigger a rename error.
	os.Remove(filepath.Join(tempDir, "Random.Movie.Name.1992.mkv"))

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

	// No files to process after removal, so we expect "Nothing to process".
	assert.NoError(t, err)
	assert.Contains(t, output.String(), "Nothing to process")
}

func Test_Movie_ProcessMovies_With_Confirm_Accept(t *testing.T) {
	tempDir := t.TempDir()
	prepareMovies(t, tempDir, []string{"Random.Movie.Name.1992.mkv"})

	movies, err := model.Movies(tempDir, []string{"mkv"}, false)
	require.NoError(t, err)
	require.Len(t, movies, 1)

	p := &mockPrompter{
		confirmResults: []mockConfirmResult{{confirmed: true}},
		inputResults: []mockInputResult{
			{value: "Random Movie Name"},
			{value: "1992"},
		},
	}

	output := new(bytes.Buffer)
	svc.Console.SetOutput(output)

	err = processMovies(context.Background(), output, tempDir, movies, os.Getuid(), os.Getgid(), p)
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(tempDir, "Random Movie Name (1992).mkv"))
}

func Test_Movie_ProcessMovies_With_Confirm_Decline(t *testing.T) {
	tempDir := t.TempDir()
	prepareMovies(t, tempDir, []string{"Random.Movie.Name.1992.mkv"})

	movies, err := model.Movies(tempDir, []string{"mkv"}, false)
	require.NoError(t, err)

	p := &mockPrompter{
		confirmResults: []mockConfirmResult{{confirmed: false}},
	}

	output := new(bytes.Buffer)
	err = processMovies(context.Background(), output, tempDir, movies, os.Getuid(), os.Getgid(), p)
	assert.NoError(t, err)

	// File should NOT be renamed since user declined.
	assert.FileExists(t, filepath.Join(tempDir, "Random.Movie.Name.1992.mkv"))
	_, err = os.Stat(filepath.Join(tempDir, "Random Movie Name (1992).mkv"))
	assert.True(t, os.IsNotExist(err))
}

func Test_Movie_ProcessMovies_With_Confirm_Interrupt(t *testing.T) {
	tempDir := t.TempDir()
	prepareMovies(t, tempDir, []string{"Random.Movie.Name.1992.mkv"})

	movies, err := model.Movies(tempDir, []string{"mkv"}, false)
	require.NoError(t, err)

	p := &mockPrompter{
		confirmResults: []mockConfirmResult{{confirmed: false, err: fmt.Errorf("interrupted")}},
	}

	output := new(bytes.Buffer)
	err = processMovies(context.Background(), output, tempDir, movies, os.Getuid(), os.Getgid(), p)
	assert.NoError(t, err)

	// File should NOT be renamed since user interrupted.
	assert.FileExists(t, filepath.Join(tempDir, "Random.Movie.Name.1992.mkv"))
}

func Test_Movie_ProcessMovies_With_Name_Override(t *testing.T) {
	tempDir := t.TempDir()
	prepareMovies(t, tempDir, []string{"Random.Movie.Name.1992.mkv"})

	movies, err := model.Movies(tempDir, []string{"mkv"}, false)
	require.NoError(t, err)

	p := &mockPrompter{
		confirmResults: []mockConfirmResult{{confirmed: true}},
		inputResults: []mockInputResult{
			{value: "Custom Title"},
			{value: "2000"},
		},
	}

	output := new(bytes.Buffer)
	svc.Console.SetOutput(output)

	err = processMovies(context.Background(), output, tempDir, movies, os.Getuid(), os.Getgid(), p)
	assert.NoError(t, err)
	assert.FileExists(t, filepath.Join(tempDir, "Custom Title (2000).mkv"))
}

func Test_Movie_ProcessMovies_With_Name_Input_Interrupt(t *testing.T) {
	tempDir := t.TempDir()
	prepareMovies(t, tempDir, []string{"Random.Movie.Name.1992.mkv"})

	movies, err := model.Movies(tempDir, []string{"mkv"}, false)
	require.NoError(t, err)

	p := &mockPrompter{
		confirmResults: []mockConfirmResult{{confirmed: true}},
		inputResults: []mockInputResult{
			{value: "", err: fmt.Errorf("interrupted")},
		},
	}

	output := new(bytes.Buffer)
	err = processMovies(context.Background(), output, tempDir, movies, os.Getuid(), os.Getgid(), p)
	assert.NoError(t, err)

	// File should NOT be renamed since user interrupted at name input.
	assert.FileExists(t, filepath.Join(tempDir, "Random.Movie.Name.1992.mkv"))
}

func Test_Movie_ProcessMovies_With_Year_Input_Interrupt(t *testing.T) {
	tempDir := t.TempDir()
	prepareMovies(t, tempDir, []string{"Random.Movie.Name.1992.mkv"})

	movies, err := model.Movies(tempDir, []string{"mkv"}, false)
	require.NoError(t, err)

	p := &mockPrompter{
		confirmResults: []mockConfirmResult{{confirmed: true}},
		inputResults: []mockInputResult{
			{value: "Random Movie Name"},
			{value: "", err: fmt.Errorf("interrupted")},
		},
	}

	output := new(bytes.Buffer)
	err = processMovies(context.Background(), output, tempDir, movies, os.Getuid(), os.Getgid(), p)
	assert.NoError(t, err)

	// File should NOT be renamed since user interrupted at year input.
	assert.FileExists(t, filepath.Join(tempDir, "Random.Movie.Name.1992.mkv"))
}

func Test_Movie_ProcessMovies_With_Invalid_Year(t *testing.T) {
	tempDir := t.TempDir()
	prepareMovies(t, tempDir, []string{"Random.Movie.Name.1992.mkv"})

	movies, err := model.Movies(tempDir, []string{"mkv"}, false)
	require.NoError(t, err)

	p := &mockPrompter{
		confirmResults: []mockConfirmResult{{confirmed: true}},
		inputResults: []mockInputResult{
			{value: "Random Movie Name"},
			{value: "not-a-number"},
		},
	}

	output := new(bytes.Buffer)
	err = processMovies(context.Background(), output, tempDir, movies, os.Getuid(), os.Getgid(), p)
	assert.NoError(t, err)

	// File should NOT be renamed since year was invalid (skipped).
	assert.FileExists(t, filepath.Join(tempDir, "Random.Movie.Name.1992.mkv"))
}

func Test_Movie_ProcessMovies_Rename_Error(t *testing.T) {
	tempDir := t.TempDir()
	prepareMovies(t, tempDir, []string{"Random.Movie.Name.1992.mkv"})

	movies, err := model.Movies(tempDir, []string{"mkv"}, false)
	require.NoError(t, err)

	// Remove the file after parsing to trigger rename error.
	os.Remove(filepath.Join(tempDir, "Random.Movie.Name.1992.mkv"))

	p := &mockPrompter{
		confirmResults: []mockConfirmResult{{confirmed: true}},
		inputResults: []mockInputResult{
			{value: "Random Movie Name"},
			{value: "1992"},
		},
	}

	output := new(bytes.Buffer)
	err = processMovies(context.Background(), output, tempDir, movies, os.Getuid(), os.Getgid(), p)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not rename")
}

func prepareMovies(t *testing.T, dir string, files []string) {
	t.Helper()
	for _, file := range files {
		os.Create(filepath.Join(dir, file))
	}
}
