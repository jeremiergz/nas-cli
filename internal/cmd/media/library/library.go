package library

import (
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/cmd/media/library/list"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/library/match"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/library/upload"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	libraryDesc = "Perform Plex library maintenance tasks"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "library",
		Aliases: []string{"lib"},
		Short:   libraryDesc,
		Long:    libraryDesc + ".",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := cmdutil.CallParentPersistentPreRunE(cmd, args)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.AddCommand(list.New())
	cmd.AddCommand(match.New())
	cmd.AddCommand(upload.New())

	return cmd
}
