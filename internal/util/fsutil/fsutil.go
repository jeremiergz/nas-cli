package fsutil

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/jeremiergz/nas-cli/internal/config"
)

// Verifies that given path exists and sets "config.WD" variable.
func InitializeWorkingDir(path string) error {
	config.WD, _ = filepath.Abs(path)
	stats, err := os.Stat(config.WD)
	if err != nil || !stats.IsDir() {
		return fmt.Errorf("%s is not a valid directory", config.WD)
	}

	return nil
}

// Lists files in directory with filter on extensions and regular expression.
func List(wd string, extensions []string, regExp *regexp.Regexp) []string {
	files, _ := os.ReadDir(wd)

	filesList := []string{}
	for _, f := range files {
		ext := strings.Replace(path.Ext(f.Name()), ".", "", 1)
		isValidExt := slices.Contains(extensions, ext)
		shouldProcess := !f.IsDir() && isValidExt
		if shouldProcess {
			if regExp == nil || !regExp.Match([]byte(f.Name())) {
				filesList = append(filesList, f.Name())
			}
		}
	}
	return filesList
}

// Creates target directory, setting its mode to 755 and setting ownership.
func PrepareDir(targetDirectory string, owner, group int) {
	os.Mkdir(targetDirectory, config.DirectoryMode)
	os.Chmod(targetDirectory, config.DirectoryMode)
	os.Chown(targetDirectory, owner, group)
}
