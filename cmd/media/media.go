package media

import (
	"fmt"
	"os/user"
	"regexp"
	"strconv"
	"strings"

	"github.com/jeremiergz/nas-cli/cmd/media/download"
	"github.com/jeremiergz/nas-cli/cmd/media/format"
	"github.com/jeremiergz/nas-cli/cmd/media/list"
	"github.com/jeremiergz/nas-cli/cmd/media/merge"
	"github.com/jeremiergz/nas-cli/cmd/media/scp"
	"github.com/jeremiergz/nas-cli/cmd/media/subsync"
	"github.com/jeremiergz/nas-cli/util/media"
	"github.com/spf13/cobra"
)

var ownershipRegexp = regexp.MustCompile(`^(\w+):?(\w+)?$`)

func NewMediaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "media",
		Short: "Set of utilities for media management",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error

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

	cmd.PersistentFlags().StringP("owner", "o", "", "override default ownership")
	cmd.AddCommand(download.NewDownloadCmd())
	cmd.AddCommand(format.NewFormatCmd())
	cmd.AddCommand(list.NewListCmd())
	cmd.AddCommand(merge.NewMergeCmd())
	cmd.AddCommand(scp.NewScpCmd())
	cmd.AddCommand(subsync.NewSubsyncCmd())

	return cmd
}
