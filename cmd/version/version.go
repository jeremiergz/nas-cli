package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/cmd/info"
)

func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print application version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(cmd.Parent().Name(), info.Version)
		},
	}

	return cmd
}
