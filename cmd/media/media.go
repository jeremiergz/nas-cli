package media

import (
	"regexp"

	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/cmd/media/download"
	"github.com/jeremiergz/nas-cli/cmd/media/format"
	"github.com/jeremiergz/nas-cli/cmd/media/list"
	"github.com/jeremiergz/nas-cli/cmd/media/merge"
	"github.com/jeremiergz/nas-cli/cmd/media/scp"
	"github.com/jeremiergz/nas-cli/cmd/media/subsync"
	"github.com/jeremiergz/nas-cli/util"
)

var (
	ownership       string
	ownershipRegexp = regexp.MustCompile(`^(\w+):?(\w+)?$`)
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "media",
		Short: "Set of utilities for media management",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := util.InitOwnership(ownership)

			return err
		},
	}

	cmd.PersistentFlags().StringVarP(&ownership, "owner", "o", "", "override default ownership")
	cmd.AddCommand(download.NewCommand())
	cmd.AddCommand(format.NewCommand())
	cmd.AddCommand(list.NewCommand())
	cmd.AddCommand(merge.NewCommand())
	cmd.AddCommand(scp.NewCommand())
	cmd.AddCommand(subsync.NewCommand())

	return cmd
}
