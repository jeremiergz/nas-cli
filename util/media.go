package util

import (
	"fmt"
	"os/user"
	"regexp"
	"strconv"
	"strings"

	"github.com/jeremiergz/nas-cli/config"
)

var (
	ownershipRegexp = regexp.MustCompile(`^(\w+):?(\w+)?$`)
)

func InitOwnership(ownership string) (err error) {
	selectedUser, _ := user.Current()
	selectedGroup := &user.Group{Gid: selectedUser.Gid}

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

	config.UID, err = strconv.Atoi(selectedUser.Uid)
	if err != nil {
		return fmt.Errorf("could not set user %s", selectedUser.Username)
	}

	config.GID, err = strconv.Atoi(selectedGroup.Gid)
	if err != nil {
		return fmt.Errorf("could not set group %s", selectedGroup.Name)
	}

	return nil
}

// Returns formatted TV show episode name from given parameters
func ToEpisodeName(title string, season int, episode int, extension string) string {
	return fmt.Sprintf("%s - S%02dE%02d.%s", title, season, episode, extension)
}

// Returns formatted movie name from given parameters
func ToMovieName(title string, year int, extension string) string {
	return fmt.Sprintf("%s (%d).%s", title, year, extension)
}

// Returns formatted season name from given parameter
func ToSeasonName(season int) string {
	return fmt.Sprintf("Season %d", season)
}
