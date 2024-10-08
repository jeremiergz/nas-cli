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

var (
	desc = "CLI application for managing my NAS"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   config.AppName,
		Short: desc,
		Long:  desc + ".",
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if debug {
				out := cmd.OutOrStdout()
				fmt.Fprintln(out)

				called := strings.Join(strings.Split(cmd.CommandPath(), " ")[1:], " ")
				fmt.Fprintln(out, "Command called:", called)
				fmt.Fprintln(out, "Execution time:", time.Since(startTime))
			}
		},
	}

	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug mode")
	cmd.PersistentFlags().IntVar(&cmdutil.MaxConcurrentGoroutines, "max-concurrent-threads", 1000, "maximum number of concurrent threads")

	return cmd
}
