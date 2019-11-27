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
		rootCmd := cmd.Parent()
		switch args[0] {
		case "bash":
			rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			rootCmd.GenZshCompletion(os.Stdout)
		}
	},
}
