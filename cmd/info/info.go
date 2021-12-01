package info

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// BuildDate is the last build datetime, overriden as ldflag
	BuildDate = "N/A"

	// Compiler is the the compiler toolchain that built the running binary
	Compiler = fmt.Sprintf("%s/%s", runtime.Compiler, runtime.Version())

	// GitCommit is the last commit SHA string, overriden as ldflag
	GitCommit = "N/A"

	// Platform is the system OS and architecture binary is built for
	Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	// Version is the Calendar Versioning string, overriden as ldflag
	Version = "N/A"
)

func NewInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Print application information",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println("BuildDate:", BuildDate)
			cmd.Println("Compiler: ", Compiler)
			cmd.Println("GitCommit:", GitCommit)
			cmd.Println("Platform: ", Platform)
			cmd.Println("Version:  ", Version)
		},
	}

	return cmd
}
