package list

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/internal/config"
)

type animeCommand struct {
	c *cobra.Command
}

var (
	animeDesc = "List animes"
)

func newAnimeCmd() *animeCommand {
	animeCmd := &animeCommand{}
	animeCmd.c = &cobra.Command{
		Use:     "animes [name]",
		Aliases: []string{"ani", "a"},
		Short:   animeDesc,
		Long:    animeDesc + ".",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			animesFolders := viper.GetStringSlice(config.KeySCPDestAnimesPaths)
			if len(animesFolders) == 0 {
				return fmt.Errorf("%s configuration entry is missing", config.KeySCPDestAnimesPaths)
			}

			var animeName string
			if len(args) > 0 {
				animeName = args[0]
			}

			err := process(cmd.Context(), animeCmd, animesFolders, animeName)
			if err != nil {
				return err
			}

			return nil
		},
	}
	return animeCmd
}

func (c *animeCommand) Command() *cobra.Command {
	return c.c
}

func (c *animeCommand) Kind() mediumKind {
	return mediumKindAnime
}

func (c *animeCommand) Out() io.Writer {
	return c.c.OutOrStdout()
}
