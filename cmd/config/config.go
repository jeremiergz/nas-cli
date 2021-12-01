package config

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure application",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			_, err = os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("could not find user home directory")
			}

			return nil
		},
	}

	cmd.AddCommand(NewGetCmd())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewSetCmd())

	return cmd
}
