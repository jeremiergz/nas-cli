package fsutil

import (
	"fmt"
	"os"
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
func List(wd string, extensions []string, regExp *regexp.Regexp, recursive bool) []string {
	fileList := []string{}
	filepath.WalkDir(wd, func(path string, entry os.DirEntry, err error) error {
		if err != nil || path == wd {
			return nil
		}

		relPath, _ := filepath.Rel(wd, path)
		if !recursive && strings.Contains(relPath, string(filepath.Separator)) {
			return filepath.SkipDir
		}

		ext := strings.Replace(filepath.Ext(path), ".", "", 1)
		isValidExt := slices.Contains(extensions, ext)
		if !isValidExt {
			return nil
		}

		if !entry.IsDir() {
			if regExp == nil || !regExp.Match([]byte(entry.Name())) {
				fileList = append(fileList, relPath)
			}
		}

		return nil
	})

	return fileList
}
