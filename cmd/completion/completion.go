package completion

import (
	"bytes"
	"fmt"
	"strings"

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
			buf := new(bytes.Buffer)
			switch args[0] {
			case "bash":
				rootCmd.GenBashCompletion(buf)

			case "zsh":
				rootCmd.GenZshCompletion(buf)
			}

			fmt.Println(strings.TrimSpace(buf.String()))

			return nil
		},
	}

	return cmd
}
