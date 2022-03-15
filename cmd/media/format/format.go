package format

import (
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/service"
	"github.com/jeremiergz/nas-cli/util"
)

func NewFormatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "format",
		Aliases: []string{"fmt"},
		Short:   "Batch media formatting depending on their type",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			mediaSvc := cmd.Context().Value(util.ContextKeyMedia).(*service.MediaService)

			err := util.CmdCallParentPersistentPreRunE(cmd, args)
			if err != nil {
				return err
			}

			return mediaSvc.InitializeWD(args[0])
		},
	}

	cmd.PersistentFlags().Bool("dry-run", false, "print result without processing it")
	cmd.PersistentFlags().StringArrayP("ext", "e", []string{"avi", "mkv", "mp4"}, "filter files by extension")
	cmd.AddCommand(NewMovieCmd())
	cmd.AddCommand(NewTVShowCmd())

	return cmd
}
