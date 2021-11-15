package media

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jeremiergz/nas-cli/cmd/media/download"
	"github.com/jeremiergz/nas-cli/cmd/media/format"
	"github.com/jeremiergz/nas-cli/cmd/media/merge"
	"github.com/jeremiergz/nas-cli/cmd/media/scp"
	"github.com/jeremiergz/nas-cli/cmd/media/subsync"
	"github.com/jeremiergz/nas-cli/util/media"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.PersistentFlags().StringP("owner", "o", "", "override default ownership")
	Cmd.AddCommand(download.Cmd)
	Cmd.AddCommand(format.Cmd)
	Cmd.AddCommand(merge.Cmd)
	Cmd.AddCommand(scp.Cmd)
	Cmd.AddCommand(subsync.Cmd)
}

var ownershipRegexp = regexp.MustCompile(`^(\w+):?(\w+)?$`)

var Cmd = &cobra.Command{
	Use:   "media",
	Short: "Set of utilities for media management",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error

		// Exit if directory retrieved from args does not exist
		media.WD, _ = filepath.Abs(args[0])
		stats, err := os.Stat(media.WD)
		if err != nil || !stats.IsDir() {
			return fmt.Errorf("%s is not a valid directory", media.WD)
		}

		selectedUser, _ := user.Current()
		selectedGroup := &user.Group{Gid: selectedUser.Gid}

		ownership, _ := cmd.Flags().GetString("owner")
		if ownership != "" {
			if !ownershipRegexp.MatchString(ownership) {
				return fmt.Errorf("ownership must be expressed as <user>[:group]")
			}

			matches := strings.Split(ownership, ":")

			userName := matches[0]
			selectedUser, err = user.Lookup(userName)
			if err != nil {
				return fmt.Errorf("could not find user %s", userName)
			}

			if len(matches) > 1 {
				groupName := userName
				if matches[1] != "" {
					groupName = matches[1]
				}
				selectedGroup, err = user.LookupGroup(groupName)
				if err != nil {
					return fmt.Errorf("could not find group %s", groupName)
				}
			}
		}

		media.UID, err = strconv.Atoi(selectedUser.Uid)
		if err != nil {
			return fmt.Errorf("could not set user %s", selectedUser.Username)
		}

		media.GID, err = strconv.Atoi(selectedGroup.Gid)
		if err != nil {
			return fmt.Errorf("could not set group %s", selectedGroup.Name)
		}

		return nil
	},
}
