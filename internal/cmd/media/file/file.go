package file

import (
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/cmd/media/file/clean"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/file/format"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/file/merge"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	fileDesc = "Manage media files"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "file",
		Aliases: []string{"f"},
		Short:   fileDesc,
		Long:    fileDesc + ".",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := cmdutil.CallParentPersistentPreRunE(cmd, args)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.AddCommand(clean.New())
	cmd.AddCommand(format.New())
	cmd.AddCommand(merge.New())

	return cmd
}
