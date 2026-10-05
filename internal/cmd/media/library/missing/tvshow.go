package missing

import (
	"github.com/spf13/cobra"
)

var (
	tvShowDesc = "List missing aired TV show episodes"
)

func newTVShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "tvshows",
		Aliases: []string{"tvshow", "tv", "t"},
		Short:   tvShowDesc,
		Long:    tvShowDesc + ".",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return processMissing(cmd, "tvshows")
		},
	}
}
