package info

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/jeremiergz/nas-cli/util/cmdutil"
	"github.com/jeremiergz/nas-cli/util/processutil"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Print details about the agent",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.OnlyValidOutputs()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			info := map[string]string{
				"buildDate": processutil.BuildDate,
				"compiler":  processutil.Compiler,
				"gitCommit": processutil.GitCommit,
				"platform":  processutil.Platform,
				"version":   processutil.Version,
			}

			w := cmd.OutOrStdout()

			switch cmdutil.OutputFormat {
			case "json":
				out, _ := json.Marshal(info)
				fmt.Fprintln(w, strings.TrimSpace(string(out)))

			case "text":
				toPrint := []string{}
				for key, value := range info {
					toPrint = append(toPrint, fmt.Sprintf("%s%-9s %s", strings.ToUpper(key[0:1]), key[1:]+":", value))
				}
				sort.Strings(toPrint)
				fmt.Fprintln(w, strings.Join(toPrint, "\n"))

			case "yaml":
				out, _ := yaml.Marshal(info)
				fmt.Fprintln(w, strings.TrimSpace(string(out)))
			}

			return nil
		},
	}

	cmdutil.AddOutputFlag(cmd)

	return cmd
}
