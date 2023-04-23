package merge

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/service"
	"github.com/jeremiergz/nas-cli/util"
	"github.com/jeremiergz/nas-cli/util/cmdutil"
)

type backup struct {
	currentPath  string
	originalPath string
}

const (
	mergeCommand string = "mkvmerge"
)

var (
	delete            bool
	dryRun            bool
	subtitleExtension string
	subtitleLanguages []string
	videoExtensions   []string
	yes               bool
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "merge",
		Aliases: []string{"mrg"},
		Short:   "Merge tracks using MKVMerge tool",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			mediaSvc := cmd.Context().Value(util.ContextKeyMedia).(*service.MediaService)

			err := cmdutil.CallParentPersistentPreRunE(cmd.Parent(), args)
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

	cmd.PersistentFlags().BoolVarP(&delete, "delete", "d", false, "delete original files")
	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print result without processing it")
	cmd.PersistentFlags().StringArrayVarP(&subtitleLanguages, "language", "l", []string{"eng", "fre"}, "language tracks to merge")
	cmd.PersistentFlags().StringVar(&subtitleExtension, "sub-ext", "srt", "subtitles extension")
	cmd.PersistentFlags().StringArrayVarP(&videoExtensions, "video-ext", "e", []string{"avi", "mkv", "mp4"}, "filter video files by extension")
	cmd.PersistentFlags().BoolVarP(&yes, "yes", "y", false, "automatic yes to prompts")
	cmd.AddCommand(newTVShowCmd())

	return cmd
}
