package version

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/jeremiergz/nas-cli/cmd/info"
)

var (
	// Cmd prints application version
	Cmd = &cobra.Command{
		Use:   "version",
		Short: "Print application version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(cmd.Parent().Name(), info.Version)
		},
	}
)
