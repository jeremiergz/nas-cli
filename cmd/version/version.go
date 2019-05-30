package version

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// BuildDate is the last build datetime
	BuildDate = ""

	// Compiler is the the compiler toolchain that built the running binary
	Compiler = runtime.Compiler

	// GitCommit is the last commit SHA string
	GitCommit = ""

	// GoVersion is the Golang compiler version
	GoVersion = runtime.Version()

	// Platform is the system OS and architecture binary is built for
	Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	// Version is the Calendar Versioning string
	Version = ""

	// VersionCmd prints application information
	VersionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print application information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Version:  ", Version)
			fmt.Println("GitCommit:", GitCommit)
			fmt.Println("BuildDate:", BuildDate)
			fmt.Println("GoVersion:", GoVersion)
			fmt.Println("Compiler: ", Compiler)
			fmt.Println("Platform: ", Platform)
		},
	}
)
