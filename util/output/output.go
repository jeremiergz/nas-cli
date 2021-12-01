package output

import (
	"fmt"

	"github.com/jeremiergz/nas-cli/util"
	"github.com/spf13/cobra"
)

var (
	Formats = []string{"json", "text", "yaml"}
	Format  string
)

func AddOutputFlag(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&Format, "output", "o", "text", "Select output format")
	cmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return Formats, cobra.ShellCompDirectiveDefault
	})
}

func OnlyValidOutputs() error {
	if !util.StringInSlice(Format, Formats) {
		return fmt.Errorf("invalid value %s for --output", Format)
	}

	return nil
}
