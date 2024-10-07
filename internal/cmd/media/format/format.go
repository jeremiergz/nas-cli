package format

import (
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/fsutil"
)

var (
	formatDesc = "Batch media formatting depending on their type"
	dryRun     bool
	extensions []string
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "format",
		Aliases: []string{"fmt"},
		Short:   formatDesc,
		Long:    formatDesc + ".",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := cmdutil.CallParentPersistentPreRunE(cmd.Parent(), args)
			if err != nil {
				return err
			}

			err = fsutil.InitializeWorkingDir(args[0])
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print result without processing it")
	cmd.PersistentFlags().StringArrayVarP(&extensions, "ext", "e", util.AcceptedVideoExtensions, "filter files by extension")
	cmd.AddCommand(newMovieCmd())
	cmd.AddCommand(newShowCmd())

	return cmd
}
