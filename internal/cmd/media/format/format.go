package format

import (
	"github.com/spf13/cobra"

	mediasvc "github.com/jeremiergz/nas-cli/internal/service/media"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
	"github.com/jeremiergz/nas-cli/internal/util/ctxutil"
)

var (
	dryRun     bool
	extensions []string
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "format",
		Aliases: []string{"fmt"},
		Short:   "Batch media formatting depending on their type",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			mediaSvc := ctxutil.Singleton[*mediasvc.Service](ctx)

			err := cmdutil.CallParentPersistentPreRunE(cmd.Parent(), args)
			if err != nil {
				return err
			}

			return mediaSvc.InitializeWD(args[0])
		},
	}

	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print result without processing it")
	cmd.PersistentFlags().StringArrayVarP(&extensions, "ext", "e", []string{"avi", "mkv", "mp4"}, "filter files by extension")
	cmd.AddCommand(newMovieCmd())
	cmd.AddCommand(newTVShowCmd())

	return cmd
}
