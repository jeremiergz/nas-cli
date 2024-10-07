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

var (
	tvShowDesc = "List TV shows"
)

func newTVShowCmd() *tvShowCommand {
	cmd := &tvShowCommand{}
	cmd.c = &cobra.Command{
		Use:     "tvshows [name]",
		Aliases: []string{"tv", "t"},
		Short:   tvShowDesc,
		Long:    tvShowDesc + ".",
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

			err := process(cmd, tvShowsFolders, tvShowName)
			if err != nil {
				return err
			}

			return nil
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
