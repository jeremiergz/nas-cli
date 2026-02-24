package subtitle

import (
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/cmd/media/subtitle/clean"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/subtitle/sync"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	subtitleDesc = "Use tools to manage subtitles"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "subtitle",
		Aliases: []string{"sub"},
		Short:   subtitleDesc,
		Long:    subtitleDesc + ".",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := cmdutil.CallParentPersistentPreRunE(cmd, args)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.AddCommand(clean.New())
	cmd.AddCommand(sync.New())

	return cmd
}
