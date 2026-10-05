package missing

import (
	"github.com/spf13/cobra"
)

var (
	animeDesc = "List missing aired anime episodes"
)

func newAnimeCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "animes",
		Aliases: []string{"anime", "ani", "a"},
		Short:   animeDesc,
		Long:    animeDesc + ".",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return processMissing(cmd, "animes")
		},
	}
}
