package processutil

import (
	"fmt"
	"runtime"
)

var (
	// Application name, overridden as ldflag.
	AppName = "nas-cli"

	// Build timestamp, overridden as ldflag.
	BuildDate = "N/A"

	// Compiler toolchain that was used to build the binary.
	Compiler = fmt.Sprintf("%s/%s", runtime.Compiler, runtime.Version())

	// Last git commit hash, overridden as ldflag.
	GitCommit = "N/A"

	// System OS and architecture the binary is built for.
	Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	// Version is the Calendar Versioning string, overridden as ldflag
	Version = "N/A"
)
