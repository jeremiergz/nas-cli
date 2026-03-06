package match

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	movieDesc = "Match movies"
)

func newMovieCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "movies",
		Aliases: []string{"movie", "m"},
		Short:   movieDesc,
		Long:    movieDesc + ".",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pterm.Info.Println("Media type not yet implemented")
			return nil
		},
	}

	return cmd
}
