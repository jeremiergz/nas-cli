package completion

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	completionDesc = "Generate completion scripts"
	validShells    = []string{"bash", "zsh"}
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:       fmt.Sprintf("completion <%s>", strings.Join(validShells, "|")),
		Short:     completionDesc,
		Long:      completionDesc + ".",
		ValidArgs: validShells,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootCmd := cmd.Parent()
			out := cmd.OutOrStdout()

			switch args[0] {
			case "bash":
				rootCmd.GenBashCompletion(out)

			case "zsh":
				rootCmd.GenZshCompletion(out)
			}

			return nil
		},
	}

	return cmd
}
