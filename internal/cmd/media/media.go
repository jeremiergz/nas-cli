package media

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/cmd/media/file"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/library"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/subtitle"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	mediaDesc = "Set of utilities for media management"
	ownership string
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "media",
		Short: mediaDesc,
		Long:  mediaDesc + ".",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmdutil.DebugMode {
				fmt.Fprintf(cmd.OutOrStdout(), "%s PersistentPreRunE\n", cmd.CommandPath())
			}

			err := cmdutil.CallParentPersistentPreRunE(cmd, args)
			if err != nil {
				return err
			}

			err = util.InitOwnership(ownership)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&ownership, "owner", "o", "", "override default ownership")
	cmd.AddCommand(file.New())
	cmd.AddCommand(library.New())
	cmd.AddCommand(subtitle.New())

	return cmd
}
