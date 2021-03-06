package info

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// BuildDate is the last build datetime, overriden as ldflag
	BuildDate = ""

	// Compiler is the the compiler toolchain that built the running binary
	Compiler = fmt.Sprintf("%s/%s", runtime.Compiler, runtime.Version())

	// GitCommit is the last commit SHA string, overriden as ldflag
	GitCommit = ""

	// Platform is the system OS and architecture binary is built for
	Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	// Version is the Calendar Versioning string, overriden as ldflag
	Version = ""

	Cmd = &cobra.Command{
		Use:   "info",
		Short: "Print application information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("BuildDate:", BuildDate)
			fmt.Println("Compiler: ", Compiler)
			fmt.Println("GitCommit:", GitCommit)
			fmt.Println("Platform: ", Platform)
			fmt.Println("Version:  ", Version)
		},
	}
)
