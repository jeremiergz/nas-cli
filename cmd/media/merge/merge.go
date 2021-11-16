package merge

import (
	"fmt"
	"os/exec"

	"github.com/jeremiergz/nas-cli/util"
	"github.com/jeremiergz/nas-cli/util/media"
	"github.com/spf13/cobra"
)

type backup struct {
	currentPath  string
	originalPath string
}

const mergeCommand string = "mkvmerge"

func init() {
	Cmd.PersistentFlags().BoolP("delete", "d", false, "delete original files")
	Cmd.PersistentFlags().Bool("dry-run", false, "print result without processing it")
	Cmd.PersistentFlags().StringArrayP("language", "l", []string{"eng", "fre"}, "language tracks to merge")
	Cmd.PersistentFlags().String("sub-ext", "srt", "subtitles extension")
	Cmd.PersistentFlags().StringArrayP("video-ext", "e", []string{"avi", "mkv", "mp4"}, "filter video files by extension")
	Cmd.AddCommand(TVShowCmd)
}

var Cmd = &cobra.Command{
	Use:   "merge",
	Short: "Merge tracks using MKVMerge tool",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		err := util.CallParentPersistentPreRunE(cmd, args)
		if err != nil {
			return err
		}

		_, err = exec.LookPath(mergeCommand)
		if err != nil {
			return fmt.Errorf("command not found: %s", mergeCommand)
		}

		return media.InitializeWD(args[0])
	},
}
