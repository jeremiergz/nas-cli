package util

import (
	"fmt"
	"os/user"
	"regexp"
	"strconv"
	"strings"

	"github.com/jeremiergz/nas-cli/internal/config"
)

type Result struct {
	Characteristics map[string]string
	IsSuccessful    bool
	Message         string
}

var (
	// List of accepted video extensions.
	AcceptedVideoExtensions = []string{"avi", "mkv", "mp4"}

	// Accepted subtitle extension.
	AcceptedSubtitleExtension = "srt"

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

var (
	countryCodeRegexp       = regexp.MustCompile(`-\w+$`)
	langDisplayNamesMapping = map[string]string{
		"eng": "English",
		"fre": "French",
		"ger": "German",
		"ita": "Italian",
		"jpn": "Japanese",
		"spa": "Spanish",
	}
)

// Returns language display flag from given language code.
func ToLanguageDisplayName(lang string, forced bool) string {
	stringsArr := []string{}
	displayName, ok := langDisplayNamesMapping[lang]
	if !ok {
		langWithoutCountryCode := countryCodeRegexp.ReplaceAllString(lang, "")
		displayName, ok = langDisplayNamesMapping[langWithoutCountryCode]
		if !ok {
			return ""
		}
	}
	stringsArr = append(stringsArr, displayName)

	if forced {
		stringsArr = append(stringsArr, "Forced")
	}

	return strings.Join(stringsArr, " ")
}

var (
	langFlagsMapping = map[string]string{
		"eng":    "🇺🇸", // English (All).
		"eng-ca": "🇨🇦", // English (Canada).
		"eng-uk": "🇬🇧", // English (UK).
		"eng-us": "🇺🇸", // English (US).
		"fre":    "🇫🇷", // French (All).
		"fre-ca": "🇨🇦", // French (Canada).
		"fre-fr": "🇫🇷", // French (France).
		"ger":    "🇩🇪", // German (All).
		"ger-at": "🇦🇹", // German (Austria).
		"ger-de": "🇩🇪", // German (Germany).
		"ita":    "🇮🇹", // Italian.
		"jpn":    "🇯🇵", // Japanese.
		"spa":    "🇪🇸", // Spanish (All).
		"spa-es": "🇪🇸", // Spanish (Mexico).
		"spa-mx": "🇲🇽", // Spanish (Mexico).
	}
)

// Returns language display flag from given language code.
func ToLanguageFlag(lang string) string {
	flag, ok := langFlagsMapping[lang]
	if !ok {
		langWithoutCountryCode := countryCodeRegexp.ReplaceAllString(lang, "")
		flag, ok = langFlagsMapping[langWithoutCountryCode]
		if !ok {
			return ""
		}
	}
	return flag
}

var (
	langRegionalsMapping = map[string]map[string]string{
		"eng": {
			"":   "eng-us", // English (Default).
			"ca": "eng-ca", // English (Canada).
			"uk": "eng-uk", // English (UK).
			"us": "eng-us", // English (US).
		},
		"fre": {
			"":   "fre-fr", // French (Default).
			"ca": "fre-ca", // French (Canada).
			"fr": "fre-fr", // French (France).
		},
		"ger": {
			"":   "ger-de", // German (Default).
			"at": "ger-at", // German (Austria).
			"de": "ger-de", // German (Germany).
		},
		"spa": {
			"":   "spa-es", // Spanish (Default).
			"es": "spa-es", // Spanish (Spain).
			"mx": "spa-mx", // Spanish (Mexico).
		},
	}
)

// Returns regionalized language code. If not found, it returns the original language code.
func ToLanguageRegionalized(lang string) string {
	countryCode := countryCodeRegexp.FindString(lang)

	langIndex, ok := langRegionalsMapping[lang]
	if !ok {
		return lang
	}

	langRegional, ok := langIndex[countryCode]
	if !ok {
		return langIndex[""]
	}

	return langRegional
}
