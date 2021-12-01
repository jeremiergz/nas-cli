package completion

import (
	"os"

	"github.com/spf13/cobra"
)

func NewCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "completion <bash|zsh>",
		Short:     "Generate completion scripts",
		ValidArgs: []string{"bash", "zsh"},
		Args:      cobra.ExactValidArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootCmd := cmd.Parent()

			switch args[0] {
			case "bash":
				rootCmd.GenBashCompletion(os.Stdout)

			case "zsh":
				rootCmd.GenZshCompletion(os.Stdout)
			}

			return nil
		},
	}

	return cmd
}
