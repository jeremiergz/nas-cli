package util

import "os"

const (
	// DirectoryMode is the default mode to apply to directories
	DirectoryMode os.FileMode = 0755

	// FileMode is the default mode to apply to files
	FileMode os.FileMode = 0644
)

// StringInSlice checks whether given string is in given array or not
func StringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}
