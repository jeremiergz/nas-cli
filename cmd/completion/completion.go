package completion

import (
	"os"

	"github.com/spf13/cobra"
)

// Cmd prints completion script for given shell
var Cmd = &cobra.Command{
	Use:       "completion <bash|zsh>",
	Short:     "Generate completion scripts",
	Args:      cobra.ExactValidArgs(1),
	ValidArgs: []string{"bash", "zsh"},
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Parent().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Parent().GenZshCompletion(os.Stdout)
		}
	},
}
