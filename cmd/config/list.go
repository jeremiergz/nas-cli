package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/jeremiergz/nas-cli/config"
	"github.com/jeremiergz/nas-cli/util/cmdutil"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configuration entries",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return cmdutil.OnlyValidOutputs()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()

			if err := viper.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); ok {
					fmt.Fprintln(w, "no configuration entries")
				} else {
					return err
				}
			} else {
				var toPrint string
				switch cmdutil.OutputFormat {
				case "json":
					out, _ := json.MarshalIndent(viper.AllSettings(), "", "  ")
					toPrint = strings.TrimSpace(string(out))

				case "text":
					values := []string{}
					for _, key := range config.OrderedKeys {
						format := "%s=%s"
						values = append(values, fmt.Sprintf(format, key, viper.GetString(key)))
					}
					toPrint = strings.Join(values, "\n")

				case "yaml":
					var buf bytes.Buffer
					encoder := yaml.NewEncoder(&buf)
					encoder.SetIndent(2)
					encoder.Encode(viper.AllSettings())
					toPrint = strings.TrimSpace(buf.String())
				}

				fmt.Fprintln(w, toPrint)
			}

			return nil
		},
	}

	cmdutil.AddOutputFlag(cmd)

	return cmd
}
