package info

import (
	"bytes"
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

			var toPrint string
			switch cmdutil.OutputFormat {
			case "json":
				out, _ := json.MarshalIndent(info, "", "  ")
				toPrint = strings.TrimSpace(string(out))

			case "text":
				values := []string{}
				for key, value := range info {
					values = append(values, fmt.Sprintf("%s%-9s %s", strings.ToUpper(key[0:1]), key[1:]+":", value))
				}
				sort.Strings(values)
				toPrint = strings.Join(values, "\n")

			case "yaml":
				var buf bytes.Buffer
				encoder := yaml.NewEncoder(&buf)
				encoder.SetIndent(2)
				encoder.Encode(info)
				toPrint = strings.TrimSpace(buf.String())
			}

			fmt.Fprintln(w, toPrint)

			return nil
		},
	}

	cmdutil.AddOutputFlag(cmd)

	return cmd
}
