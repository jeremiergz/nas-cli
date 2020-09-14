package media

import (
	"fmt"
	"os/user"
	"strconv"

	"github.com/jeremiergz/nas-cli/cmd/media/download"
	"github.com/jeremiergz/nas-cli/cmd/media/format"
	"github.com/jeremiergz/nas-cli/cmd/media/subsync"
	"github.com/jeremiergz/nas-cli/util/media"
	"github.com/spf13/cobra"
)

func init() {
	Cmd.PersistentFlags().StringP("group", "g", "", "override default file group")
	Cmd.PersistentFlags().StringP("user", "u", "", "override default file owner")
	Cmd.AddCommand(download.Cmd)
	Cmd.AddCommand(format.Cmd)
	Cmd.AddCommand(subsync.Cmd)
}

// Cmd loads sub-commands for media management
var Cmd = &cobra.Command{
	Use:   "media",
	Short: "Set of utilities for media management",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Set user from process user or from command flag
		var err error
		owner, _ := user.Current()
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
