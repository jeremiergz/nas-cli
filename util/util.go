package util

import (
	"io"
	"net/http"
	"os"
)

const (
	// DirectoryMode is the default mode to apply to directories
	DirectoryMode os.FileMode = 0755

	// ExecutableMode is the default mode for executable files
	ExecutableMode os.FileMode = 0755

	// FileMode is the default mode to apply to files
	FileMode os.FileMode = 0644
)

// DownloadFile downloads given URL to a local path
func DownloadFile(filePath string, url string) error {
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, res.Body)
	return err
}

// StringInSlice checks whether given string is in given array or not
func StringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}
