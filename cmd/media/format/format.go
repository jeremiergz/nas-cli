package format

import (
	"github.com/spf13/cobra"
)

func init() {
	Cmd.PersistentFlags().Bool("dry-run", false, "print result without processing it")
	Cmd.PersistentFlags().StringArrayP("ext", "e", []string{"avi", "mkv", "mp4"}, "filter files by extension")
	Cmd.AddCommand(MovieCmd)
	Cmd.AddCommand(TVShowCmd)
}

var Cmd = &cobra.Command{
	Use:   "format",
	Short: "Batch media formatting depending on their type",
}
