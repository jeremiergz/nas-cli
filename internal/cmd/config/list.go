package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/jeremiergz/nas-cli/internal/config"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	listDesc = "List configuration entries"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: listDesc,
		Long:  listDesc + ".",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmdutil.DebugMode {
				fmt.Fprintf(cmd.OutOrStdout(), "%s PreRunE\n", cmd.CommandPath())
			}

			err := cmdutil.OnlyValidOutputs()
			if err != nil {
				return err
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			if err := viper.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
					return err
				}

				fmt.Fprintln(out, "no configuration entries")
				return nil
			}

			var toPrint string
			switch cmdutil.OutputFormat {
			case "json":
				out, _ := json.MarshalIndent(viper.AllSettings(), "", "  ")
				toPrint = strings.TrimSpace(string(out))

			case "text":
				values := []string{}
				for _, key := range config.OrderedKeys {
					format := "%s=%v"
					values = append(values, fmt.Sprintf(format, key, viper.Get(key)))
				}
				toPrint = strings.Join(values, "\n")

			case "yaml":
				var buf bytes.Buffer
				encoder := yaml.NewEncoder(&buf)
				encoder.SetIndent(2)
				encoder.Encode(viper.AllSettings())
				toPrint = strings.TrimSpace(buf.String())
			}

			fmt.Fprintln(out, toPrint)

			return nil
		},
	}

	cmdutil.AddOutputFlag(cmd)

	return cmd
}
