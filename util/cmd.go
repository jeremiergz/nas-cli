package util

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// All possible --output values
	CmdOutputFormats = []string{"json", "text", "yaml"}

	// The selected or default --output value
	CmdOutputFormat string
)

// Adds --output to given command
func CmdAddOutputFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&CmdOutputFormat, "output", "o", "text", "Select output format")
	cmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return CmdOutputFormats, cobra.ShellCompDirectiveDefault
	})
}

// Runs ParentPersistentPreRun if defined
func CmdCallParentPersistentPreRun(cmd *cobra.Command, args []string) {
	if parent := cmd.Parent(); parent != nil {
		if parent.PersistentPreRun != nil {
			parent.PersistentPreRun(parent, args)
		}
	}
}

// Runs ParentPersistentPreRunE if defined
func CmdCallParentPersistentPreRunE(cmd *cobra.Command, args []string) error {
	if parent := cmd.Parent(); parent != nil {
		if parent.PersistentPreRunE != nil {
			return parent.PersistentPreRunE(parent.Root(), args)
		}
	}

	return nil
}

// Checks that value passed to --output is valid
func CmdOnlyValidOutputs() error {
	if !Contains(CmdOutputFormats, CmdOutputFormat) {
		return fmt.Errorf("invalid value %s for --output", CmdOutputFormat)
	}

	return nil
}
