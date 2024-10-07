package info

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	infoDesc = "Print details about the application"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: infoDesc,
		Long:  infoDesc + ".",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			err := cmdutil.OnlyValidOutputs()
			if err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			info := map[string]string{
				"buildDate": config.BuildDate,
				"compiler":  config.Compiler,
				"gitCommit": config.GitCommit,
				"platform":  config.Platform,
				"version":   config.Version,
			}

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
