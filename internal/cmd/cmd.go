package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	debug bool

	startTime = time.Now()
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   config.AppName,
		Short: "CLI application for managing my NAS",
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if debug {
				fmt.Fprintln(cmd.OutOrStdout())

				called := strings.Join(strings.Split(cmd.CommandPath(), " ")[1:], " ")
				fmt.Fprintln(cmd.OutOrStdout(), "Command called:", called)
				fmt.Fprintln(cmd.OutOrStdout(), "Execution time:", time.Since(startTime))
			}
		},
	}

	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug mode")
	cmd.PersistentFlags().IntVar(&cmdutil.MaxConcurrentGoroutines, "max-concurrent-threads", 1000, "maximum number of concurrent threads")

	return cmd
}
