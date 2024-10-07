package config

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	configDesc = "Configure application"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: configDesc,
		Long:  configDesc + ".",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.UserHomeDir(); err != nil {
				return fmt.Errorf("could not find user home directory")
			}

			return nil
		},
	}

	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newSetCmd())

	return cmd
}
