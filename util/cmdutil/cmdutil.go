package cmdutil

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/util/sliceutil"
)

var (
	// All possible --output values
	OutputFormats = []string{"json", "text", "yaml"}

	// The selected or default --output value
	OutputFormat string
)

// Adds --output to given command
func AddOutputFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&OutputFormat, "output", "o", "text", "Select output format")
	cmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return OutputFormats, cobra.ShellCompDirectiveDefault
	})
}

// Runs ParentPersistentPreRun if defined
func CallParentPersistentPreRun(cmd *cobra.Command, args []string) {
	if parent := cmd.Parent(); parent != nil {
		if parent.PersistentPreRun != nil {
			parent.PersistentPreRun(parent, args)
		}
	}
}

// Runs ParentPersistentPreRunE if defined
func CallParentPersistentPreRunE(cmd *cobra.Command, args []string) error {
	if parent := cmd.Parent(); parent != nil {
		if parent.PersistentPreRunE != nil {
			return parent.PersistentPreRunE(parent, args)
		}
	}

	return nil
}

// Checks that value passed to --output is valid
func OnlyValidOutputs() error {
	if !sliceutil.Contains(OutputFormats, OutputFormat) {
		return fmt.Errorf("invalid value %s for --output", OutputFormat)
	}

	return nil
}
