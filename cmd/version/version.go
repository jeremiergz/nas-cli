package version

import (
	"fmt"

	"github.com/jeremiergz/nas-cli/cmd/info"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Print application version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(cmd.Parent().Name(), info.Version)
	},
}
