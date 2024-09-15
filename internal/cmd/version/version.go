package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/config"
)

var (
	versionDesc = "Print application version"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: versionDesc,
		Long:  versionDesc + ".",
		Run: func(cmd *cobra.Command, args []string) {
			w := cmd.OutOrStdout()

			fmt.Fprintln(w, cmd.Parent().Name(), config.Version)
		},
	}

	return cmd
}
