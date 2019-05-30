package completion

import (
	"os"

	"github.com/spf13/cobra"
)

// CompletionCmd prints completion script for given shell
var CompletionCmd = &cobra.Command{
	Use:       "completion <bash|zsh>",
	Short:     "Generate completion scripts",
	Args:      cobra.ExactValidArgs(1),
	ValidArgs: []string{"bash", "zsh"},
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.GenZshCompletion(os.Stdout)
		}
	},
}
