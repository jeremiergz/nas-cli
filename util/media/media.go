package media

import (
	"io/ioutil"
	"path"
	"regexp"
	"strings"

	"gitlab.com/jeremiergz/nas-cli/util"
)

var (
	// GID is the processed files group to set
	GID int
	// UID is the processed files owner to set
	UID int
	// WD is the working directory's absolute path
	WD string
)

// List lists files in directory with filter on extensions and RegExp
func List(wd string, extensions []string, regExp *regexp.Regexp) []string {
	files, _ := ioutil.ReadDir(wd)
	filesList := []string{}
	for _, f := range files {
		ext := strings.Replace(path.Ext(f.Name()), ".", "", 1)
		isValidExt := util.StringInSlice(ext, extensions)
		shouldProcess := !f.IsDir() && isValidExt && !regExp.Match([]byte(f.Name()))
		if shouldProcess {
			filesList = append(filesList, f.Name())
		}
	}
	return filesList
}
