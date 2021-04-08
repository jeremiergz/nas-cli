package config

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	configDir string
)

func init() {
	Cmd.AddCommand(GetCmd)
	Cmd.AddCommand(ListCmd)
	Cmd.AddCommand(SetCmd)
}

var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Configure application",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		configDir, err = os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not find user home directory")
		}
		return nil
	},
}
