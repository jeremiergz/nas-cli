package config

import (
	"fmt"
	"os"

	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
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
			if cmdutil.DebugMode {
				fmt.Fprintf(cmd.OutOrStdout(), "%s PersistentPreRunE\n", cmd.CommandPath())
			}

			err := cmdutil.CallParentPersistentPreRunE(cmd, args)
			if err != nil {
				return err
			}

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
