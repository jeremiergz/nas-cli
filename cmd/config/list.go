package config

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/jeremiergz/nas-cli/util/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configuration entries",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return output.OnlyValidOutputs()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); ok {
					fmt.Println("no configuration entries")
				} else {
					return err
				}
			} else {
				switch output.Format {
				case "json":
					out, _ := json.Marshal(viper.AllSettings())
					fmt.Println(strings.TrimSpace(string(out)))

				case "text":
					keys := viper.AllKeys()
					toPrint := []string{}
					for _, key := range keys {
						format := "%s=%s"
						toPrint = append(toPrint, fmt.Sprintf(format, key, viper.GetString(key)))
					}
					sort.Strings(toPrint)
					fmt.Println(strings.Join(toPrint, "\n"))

				case "yaml":
					out, _ := yaml.Marshal(viper.AllSettings())
					fmt.Println(strings.TrimSpace(string(out)))
				}
			}

			return nil
		},
	}

	output.AddOutputFlag(cmd)

	return cmd
}
