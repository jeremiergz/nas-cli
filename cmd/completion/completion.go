package completion

import (
	"os"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:       "completion <bash|zsh>",
	Short:     "Generate completion scripts",
	ValidArgs: []string{"bash", "zsh"},
	Args:      cobra.ExactValidArgs(1),
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
