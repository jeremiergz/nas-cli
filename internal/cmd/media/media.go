package media

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jeremiergz/nas-cli/internal/cmd/media/clean"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/format"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/list"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/merge"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/scp"
	"github.com/jeremiergz/nas-cli/internal/cmd/media/subsync"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

var (
	mediaDesc       = "Set of utilities for media management"
	languageRegions []string
	ownership       string

	langRegionRegexp = regexp.MustCompile(`^[a-z]{3}=[a-z]{2}-[a-z]{2}$`)
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "media",
		Short: mediaDesc,
		Long:  mediaDesc + ".",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmdutil.DebugMode {
				fmt.Fprintf(cmd.OutOrStdout(), "%s PersistentPreRunE\n", cmd.CommandPath())
			}

			err := cmdutil.CallParentPersistentPreRunE(cmd, args)
			if err != nil {
				return err
			}

			if len(languageRegions) > 0 {
				flag := cmd.Flag("lang-region")
				for _, region := range languageRegions {
					isValid := langRegionRegexp.MatchString(region)
					if !isValid {
						flagNames := []string{}
						if flag.Shorthand != "" {
							flagNames = append(flagNames, fmt.Sprintf("-%s", flag.Shorthand))
						}
						flagNames = append(flagNames, fmt.Sprintf("--%s", flag.Name))
						flagStr := strings.Join(flagNames, ", ")
						return fmt.Errorf(`invalid argument %q for %q flag: expected format is "lang=region"`, region, flagStr)
					}
					parts := strings.Split(region, "=")
					lang := parts[0]
					region := parts[1]
					util.SetDefaultLanguageRegion(lang, region)
				}
			}

			err = util.InitOwnership(ownership)
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringArrayVar(&languageRegions, "lang-region", nil, "override default language regions")
	cmd.PersistentFlags().StringVarP(&ownership, "owner", "o", "", "override default ownership")
	cmd.AddCommand(clean.New())
	cmd.AddCommand(format.New())
	cmd.AddCommand(list.New())
	cmd.AddCommand(merge.New())
	cmd.AddCommand(scp.New())
	cmd.AddCommand(subsync.New())

	return cmd
}
