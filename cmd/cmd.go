package cmd

import (
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nas-cli",
		Short: "CLI application for managing my NAS",
	}

	return cmd
}
