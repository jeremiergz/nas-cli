package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/config"
)

var (
	versionDesc = "Print application version"
	flagShort   bool
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: versionDesc,
		Long:  versionDesc + ".",
		Run: func(cmd *cobra.Command, args []string) {
			toPrint := config.Version
			if !flagShort {
				toPrint = fmt.Sprintf("%s %s", cmd.Parent().Name(), config.Version)
			}

			fmt.Fprintln(cmd.OutOrStdout(), toPrint)
		},
	}

	cmd.PersistentFlags().BoolVarP(&flagShort, "short", "s", false, "only print version number")

	return cmd
}
