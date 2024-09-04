package list

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/internal/config"
)

type tvShowCommand struct {
	c *cobra.Command
}

func newTVShowCmd() *tvShowCommand {
	cmd := &tvShowCommand{}
	cmd.c = &cobra.Command{
		Use:     "tvshows [name]",
		Aliases: []string{"tv", "t"},
		Short:   "TV Shows listing",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			tvShowsFolders := viper.GetStringSlice(config.KeySCPDestTVShowsPaths)
			if len(tvShowsFolders) == 0 {
				return fmt.Errorf("%s configuration entry is missing", config.KeySCPDestTVShowsPaths)
			}

			var tvShowName string
			if len(args) > 0 {
				tvShowName = args[0]
			}

			return process(cmd, tvShowsFolders, tvShowName)
		},
	}
	return cmd
}

func (c *tvShowCommand) Command() *cobra.Command {
	return c.c
}

func (c *tvShowCommand) Kind() mediumKind {
	return mediumKindTVShow
}

func (c *tvShowCommand) Out() io.Writer {
	return c.c.OutOrStdout()
}
