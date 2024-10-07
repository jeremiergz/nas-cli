package list

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/jeremiergz/nas-cli/internal/config"
)

type movieCommand struct {
	c *cobra.Command
}

var (
	movieDesc = "List movies"
)

func newMovieCmd() *movieCommand {
	cmd := &movieCommand{}
	cmd.c = &cobra.Command{
		Use:     "movies [name]",
		Aliases: []string{"mov", "m"},
		Short:   movieDesc,
		Long:    movieDesc + ".",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			moviesFolders := viper.GetStringSlice(config.KeySCPDestMoviesPaths)
			if len(moviesFolders) == 0 {
				return fmt.Errorf("%s configuration entry is missing", config.KeySCPDestMoviesPaths)
			}

			var movieName string
			if len(args) > 0 {
				movieName = args[0]
			}

			err := process(cmd, moviesFolders, movieName)
			if err != nil {
				return err
			}

			return nil
		},
	}
	return cmd
}

func (c *movieCommand) Command() *cobra.Command {
	return c.c
}

func (c *movieCommand) Kind() mediumKind {
	return mediumKindMovie
}

func (c *movieCommand) Out() io.Writer {
	return c.c.OutOrStdout()
}
