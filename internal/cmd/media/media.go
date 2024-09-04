package media

import (
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/cmd/media/clean"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/download"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/format"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/list"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/merge"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/scp"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/subsync"
	"github.com/jeremiergz/nas-cli/internal/util"
)

var (
	ownership string
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "media",
		Short: "Set of utilities for media management",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := util.InitOwnership(ownership)

			return err
		},
	}

	cmd.PersistentFlags().StringVarP(&ownership, "owner", "o", "", "override default ownership")
	cmd.AddCommand(clean.New())
	cmd.AddCommand(download.New())
	cmd.AddCommand(format.New())
	cmd.AddCommand(list.New())
	cmd.AddCommand(merge.New())
	cmd.AddCommand(scp.New())
	cmd.AddCommand(subsync.New())

	return cmd
}
