package clean

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"

	mediasvc "github.com/jeremiergz/nas-cli/internal/service/media"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/ctxutil"
)

var (
	dryRun            bool
	subtitleExtension string
	subtitleLanguages []string
	videoExtensions   []string
	yes               bool
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clean",
		Aliases: []string{"cln"},
		Short:   "Clean tracks using MKVPropEdit tool",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			mediaSvc := ctxutil.Singleton[*mediasvc.Service](ctx)

			err := cmdutil.CallParentPersistentPreRunE(cmd.Parent(), args)
			if err != nil {
				return err
			}

			_, err = exec.LookPath(cmdutil.CommandMKVPropEdit)
			if err != nil {
				return fmt.Errorf("command not found: %s", cmdutil.CommandMKVPropEdit)
			}

			return mediaSvc.InitializeWD(args[0])
		},
	}

	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print result without processing it")
	cmd.PersistentFlags().StringArrayVarP(&subtitleLanguages, "language", "l", []string{"eng", "fre"}, "language tracks to merge")
	cmd.PersistentFlags().StringVar(&subtitleExtension, "sub-ext", "srt", "subtitles extension")
	cmd.PersistentFlags().StringArrayVarP(&videoExtensions, "video-ext", "e", []string{"avi", "mkv", "mp4"}, "filter video files by extension")
	cmd.PersistentFlags().BoolVarP(&yes, "yes", "y", false, "automatic yes to prompts")
	cmd.AddCommand(newShowCmd())

	return cmd
}
