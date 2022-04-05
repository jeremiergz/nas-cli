package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/jeremiergz/nas-cli/config"
	"github.com/jeremiergz/nas-cli/util"
)

func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configuration entries",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return util.CmdOnlyValidOutputs()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); ok {
					fmt.Println("no configuration entries")
				} else {
					return err
				}
			} else {
				switch util.CmdOutputFormat {
				case "json":
					out, _ := json.Marshal(viper.AllSettings())
					fmt.Println(strings.TrimSpace(string(out)))

				case "text":
					toPrint := []string{}
					for _, key := range config.Keys {
						format := "%s=%s"
						toPrint = append(toPrint, fmt.Sprintf(format, key, viper.GetString(key)))
					}
					fmt.Println(strings.Join(toPrint, "\n"))

				case "yaml":
					out, _ := yaml.Marshal(viper.AllSettings())
					fmt.Println(strings.TrimSpace(string(out)))
				}
			}

			return nil
		},
	}

	util.CmdAddOutputFlag(cmd)

	return cmd
}
