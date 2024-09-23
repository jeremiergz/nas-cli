package merge

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

type backup struct {
	currentPath  string
	originalPath string
}

var (
	mergeDesc         = "Merge tracks using MKVMerge tool"
	delete            bool
	dryRun            bool
	subtitleExtension string
	subtitleLanguages []string
	videoExtensions   []string
	yes               bool
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "merge",
		Aliases: []string{"mrg"},
		Short:   mergeDesc,
		Long:    mergeDesc + ".",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := cmdutil.CallParentPersistentPreRunE(cmd.Parent(), args)
			if err != nil {
				return err
			}

			_, err = exec.LookPath(cmdutil.CommandMKVMerge)
			if err != nil {
				return fmt.Errorf("command not found: %s", cmdutil.CommandMKVMerge)
			}

			return fsutil.InitializeWorkingDir(args[0])
		},
	}

	cmd.PersistentFlags().BoolVarP(&delete, "delete", "d", false, "delete original files")
	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print result without processing it")
	cmd.PersistentFlags().StringArrayVarP(&subtitleLanguages, "language", "l", []string{"eng", "fre"}, "language tracks to merge")
	cmd.PersistentFlags().StringVar(&subtitleExtension, "sub-ext", util.AcceptedSubtitleExtension, "filter subtitles by extension")
	cmd.PersistentFlags().StringArrayVarP(&videoExtensions, "video-ext", "e", util.AcceptedVideoExtensions, "filter video files by extension")
	cmd.PersistentFlags().BoolVarP(&yes, "yes", "y", false, "automatic yes to prompts")
	cmd.AddCommand(newShowCmd())

	return cmd
}
