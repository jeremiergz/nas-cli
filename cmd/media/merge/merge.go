package merge

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/service"
	"github.com/jeremiergz/nas-cli/util"
)

type backup struct {
	currentPath  string
	originalPath string
}

const mergeCommand string = "mkvmerge"

func NewMergeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "merge",
		Aliases: []string{"mrg"},
		Short:   "Merge tracks using MKVMerge tool",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			mediaSvc := cmd.Context().Value(util.ContextKeyMedia).(*service.MediaService)

			err := util.CmdCallParentPersistentPreRunE(cmd, args)
			if err != nil {
				return err
			}

			_, err = exec.LookPath(mergeCommand)
			if err != nil {
				return fmt.Errorf("command not found: %s", mergeCommand)
			}

			return mediaSvc.InitializeWD(args[0])
		},
	}

	cmd.PersistentFlags().BoolP("delete", "d", false, "delete original files")
	cmd.PersistentFlags().Bool("dry-run", false, "print result without processing it")
	cmd.PersistentFlags().StringArrayP("language", "l", []string{"eng", "fre"}, "language tracks to merge")
	cmd.PersistentFlags().String("sub-ext", "srt", "subtitles extension")
	cmd.PersistentFlags().StringArrayP("video-ext", "e", []string{"avi", "mkv", "mp4"}, "filter video files by extension")
	cmd.PersistentFlags().BoolP("yes", "y", false, "automatic yes to prompts")
	cmd.AddCommand(NewTVShowCmd())

	return cmd
}
