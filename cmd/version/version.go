package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/util/processutil"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print application version",
		Run: func(cmd *cobra.Command, args []string) {
			w := cmd.OutOrStdout()

			fmt.Fprintln(w, cmd.Parent().Name(), processutil.GitCommit)
		},
	}

	return cmd
}
