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
	RunE: func(cmd *cobra.Command, args []string) error {
		rootCmd := cmd.Parent()
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		}

		return nil
	},
}
