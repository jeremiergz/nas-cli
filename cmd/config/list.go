package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configuration entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				fmt.Println("no configuration entries")
			} else {
				return err
			}
		} else {
			keys := viper.AllKeys()
			sort.Strings(keys)
			toPrint := []string{}
			for index, key := range keys {
				format := "%s=%s\n"
				if index == len(keys)-1 {
					format = "%s=%s"
				}
				toPrint = append(toPrint, fmt.Sprintf(format, key, viper.GetString(key)))
			}
			fmt.Println(strings.Join(toPrint, ""))
		}
		return nil
	},
}
