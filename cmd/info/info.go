package info

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/jeremiergz/nas-cli/util/output"
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
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return output.OnlyValidOutputs()
		},
		Run: func(cmd *cobra.Command, args []string) {
			info := map[string]string{
				"buildDate": BuildDate,
				"compiler":  Compiler,
				"gitCommit": GitCommit,
				"platform":  Platform,
				"version":   Version,
			}

			switch output.Format {
			case "json":
				out, _ := json.Marshal(info)
				fmt.Println(strings.TrimSpace(string(out)))

			case "text":
				for key, value := range info {
					fmt.Printf("%-10s %s\n", strings.Title(key)+":", value)
				}

			case "yaml":
				out, _ := yaml.Marshal(info)
				fmt.Println(strings.TrimSpace(string(out)))
			}
		},
	}

	output.AddOutputFlag(cmd)

	return cmd
}
