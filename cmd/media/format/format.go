package format

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	"gitlab.com/jeremiergz/nas-cli/cmd/media/format/movie"
	"gitlab.com/jeremiergz/nas-cli/cmd/media/format/tvshow"
	"gitlab.com/jeremiergz/nas-cli/util/media"
)

func init() {
	FormatCmd.PersistentFlags().Bool("dry-run", false, "print result without processing it")
	FormatCmd.PersistentFlags().StringArrayP("ext", "e", []string{"avi", "mkv", "mp4"}, "set extensions to look for in directory")
	FormatCmd.PersistentFlags().StringP("group", "g", "", "set formatted files group")
	FormatCmd.PersistentFlags().StringP("user", "u", "", "set formatted files owner")
	FormatCmd.AddCommand(movie.FormatMoviesCmd)
	FormatCmd.AddCommand(tvshow.FormatTVShowsCmd)
}

// FormatCmd formats given media type according to personal conventions
var FormatCmd = &cobra.Command{
	Use:   "format",
	Short: "Batch media formatting depending on their type",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Exit if directory retrieved from args does not exist
		if len(args) > 0 {
			media.WD, _ = filepath.Abs(args[0])
			stats, err := os.Stat(media.WD)
			if err != nil || !stats.IsDir() {
				return fmt.Errorf("%s is not a valid directory", media.WD)
			}
		}

		// Set user from process user or from command flag
		owner, err := user.Current()
		ownerName, _ := cmd.Flags().GetString("user")
		if ownerName != "" {
			owner, err = user.Lookup(ownerName)
			if err != nil {
				return fmt.Errorf("could not find user %s", ownerName)
			}
		}
		media.UID, err = strconv.Atoi(owner.Uid)
		if err != nil {
			return fmt.Errorf("could not set user %s", ownerName)
		}

		// Set group from user or from command flag
		group := &user.Group{Gid: owner.Gid}
		groupName, _ := cmd.Flags().GetString("group")
		if groupName != "" {
			group, err = user.LookupGroup(groupName)
			if err != nil {
				return fmt.Errorf("could not find group %s", groupName)
			}
		}
		media.GID, err = strconv.Atoi(group.Gid)
		if err != nil {
			return fmt.Errorf("could not set group %s", groupName)
		}

		return nil
	},
}
