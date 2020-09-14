package format

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeremiergz/nas-cli/cmd/media/format/movie"
	"github.com/jeremiergz/nas-cli/cmd/media/format/tvshow"
	"github.com/jeremiergz/nas-cli/util/media"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.PersistentFlags().Bool("dry-run", false, "print result without processing it")
	Cmd.PersistentFlags().StringArrayP("ext", "e", []string{"avi", "mkv", "mp4"}, "filter files by extension")
	Cmd.AddCommand(movie.Cmd)
	Cmd.AddCommand(tvshow.Cmd)
}

// Cmd formats given media type according to personal conventions
var Cmd = &cobra.Command{
	Use:   "format",
	Short: "Batch media formatting depending on their type",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Exit if directory retrieved from args does not exist
		media.WD, _ = filepath.Abs(args[0])
		stats, err := os.Stat(media.WD)
		if err != nil || !stats.IsDir() {
			return fmt.Errorf("%s is not a valid directory", media.WD)
		}
		return nil
	},
}
