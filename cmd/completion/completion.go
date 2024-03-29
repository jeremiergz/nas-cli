package completion

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	validShells = []string{"bash", "zsh"}
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:       fmt.Sprintf("completion <%s>", strings.Join(validShells, "|")),
		Short:     "Generate completion scripts",
		ValidArgs: validShells,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootCmd := cmd.Parent()

			w := cmd.OutOrStdout()

			switch args[0] {
			case "bash":
				rootCmd.GenBashCompletion(w)

			case "zsh":
				rootCmd.GenZshCompletion(w)
			}

			return nil
		},
	}

	return cmd
}
