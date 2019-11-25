package media

import (
	"path"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/jeremiergz/nas-cli/cmd/media/format"
	"gitlab.com/jeremiergz/nas-cli/util"
)

func init() {
	Cmd.AddCommand(format.Cmd)
}

// Cmd loads sub-commands for media management
var Cmd = &cobra.Command{
	Use:   "media",
	Short: "Set of utilities for media management",
}

// filterByExtensions filters given array against valid extensions array
func filterByExtensions(paths []string, extensions []string) []string {
	filteredPaths := make([]string, 0)
	for _, p := range paths {
		ext := strings.Replace(path.Ext(p), ".", "", 1)
		isValid := util.StringInSlice(ext, extensions)
		if isValid {
			filteredPaths = append(filteredPaths, p)
		}
	}

	return filteredPaths
}
