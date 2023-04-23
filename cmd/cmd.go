package cmd

import (
	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/util/processutil"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   processutil.AppName,
		Short: "CLI application for managing my NAS",
	}

	return cmd
}
