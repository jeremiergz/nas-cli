package match

import (
	"github.com/spf13/cobra"

	svc "github.com/jeremiergz/nas-cli/internal/service"
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
			svc.Console.Info("Media type not yet implemented")
			return nil
		},
	}

	return cmd
}
