package media

import (
	"fmt"
	"os/user"
	"path"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/jeremiergz/nas-cli/cmd/media/download"
	"gitlab.com/jeremiergz/nas-cli/cmd/media/format"
	"gitlab.com/jeremiergz/nas-cli/cmd/media/subsync"
	"gitlab.com/jeremiergz/nas-cli/util"
	"gitlab.com/jeremiergz/nas-cli/util/media"
)

func init() {
	Cmd.PersistentFlags().StringP("group", "g", "", "override default file group")
	Cmd.PersistentFlags().StringP("user", "u", "", "override default file owner")
	Cmd.AddCommand(download.Cmd)
	Cmd.AddCommand(format.Cmd)
	Cmd.AddCommand(subsync.Cmd)
}

// filterByExtensions filters given array against valid extensions array
func filterByExtensions(paths []string, extensions []string) []string {
	filteredPaths := make([]string, 0)
	for _, p := range paths {
		ext := strings.Replace(path.Ext(p), ".", "", 1)
		isValid := util.StringInSlice(ext, extensions)
		if isValid {
			filteredPaths = append(filteredPaths, p)
		}
	}
	return filteredPaths
}

// Cmd loads sub-commands for media management
var Cmd = &cobra.Command{
	Use:   "media",
	Short: "Set of utilities for media management",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
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
