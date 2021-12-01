package format

import (
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/util"
	"github.com/jeremiergz/nas-cli/util/media"
)

func NewFormatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "format",
		Short: "Batch media formatting depending on their type",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := util.CallParentPersistentPreRunE(cmd, args)
			if err != nil {
				return err
			}

			return media.InitializeWD(args[0])
		},
	}

	cmd.PersistentFlags().Bool("dry-run", false, "print result without processing it")
	cmd.PersistentFlags().StringArrayP("ext", "e", []string{"avi", "mkv", "mp4"}, "filter files by extension")
	cmd.AddCommand(NewMovieCmd())
	cmd.AddCommand(NewTVShowCmd())

	return cmd
}
