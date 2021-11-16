package merge

import (
	"fmt"
	"os/exec"

	"github.com/jeremiergz/nas-cli/util/media"
	"github.com/spf13/cobra"
)

type backup struct {
	currentPath  string
	originalPath string
}

type subtitles map[string]map[string]string

const mergeCommand string = "mkvmerge"

func init() {
	Cmd.PersistentFlags().BoolP("keep", "k", true, "keep original files")
	Cmd.PersistentFlags().StringArrayP("language", "l", []string{"eng", "fre"}, "language tracks to merge")
	Cmd.PersistentFlags().StringP("name", "n", "", "override name")
	Cmd.PersistentFlags().String("sub-ext", "srt", "subtitles extension")
	Cmd.PersistentFlags().StringArrayP("video-ext", "e", []string{"avi", "mkv", "mp4"}, "filter video files by extension")
	Cmd.AddCommand(TVShowCmd)
}

var Cmd = &cobra.Command{
	Use:   "merge",
	Short: "Merge tracks using MKVMerge tool",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		_, err := exec.LookPath(mergeCommand)
		if err != nil {
			return fmt.Errorf("command not found: %s", mergeCommand)
		}

		return media.InitializeWD(args[0])
	},
}
