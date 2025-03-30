package util

import (
	"fmt"
	"os/user"
	"regexp"
	"strconv"
	"strings"

	"github.com/jeremiergz/nas-cli/internal/config"
)

const (
	ExtensionAVI string = "avi"
	ExtensionMKV string = "mkv"
	ExtensionMP4 string = "mp4"
)

var (
	// List of accepted video extensions.
	AcceptedVideoExtensions = []string{ExtensionAVI, ExtensionMKV, ExtensionMP4}

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
		"eng-gb": "🇬🇧", // English (UK).
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
		"en": {
			"":   "eng-us", // English (Default).
			"ca": "eng-ca", // English (Canada).
			"gb": "eng-gb", // English (UK).
			"us": "eng-us", // English (US).
		},
		"eng": {
			"":   "eng-us", // English (Default).
			"ca": "eng-ca", // English (Canada).
			"gb": "eng-gb", // English (UK).
			"us": "eng-us", // English (US).
		},
		"fr": {
			"":   "fre-fr", // French (Default).
			"ca": "fre-ca", // French (Canada).
			"fr": "fre-fr", // French (France).
		},
		"fre": {
			"":   "fre-fr", // French (Default).
			"ca": "fre-ca", // French (Canada).
			"fr": "fre-fr", // French (France).
		},
		"de": {
			"":   "ger-de", // German (Default).
			"at": "ger-at", // German (Austria).
			"de": "ger-de", // German (Germany).
		},
		"ger": {
			"":   "ger-de", // German (Default).
			"at": "ger-at", // German (Austria).
			"de": "ger-de", // German (Germany).
		},
		"es": {
			"":   "spa-es", // Spanish (Default).
			"es": "spa-es", // Spanish (Spain).
			"mx": "spa-mx", // Spanish (Mexico).
		},
		"spa": {
			"":   "spa-es", // Spanish (Default).
			"es": "spa-es", // Spanish (Spain).
			"mx": "spa-mx", // Spanish (Mexico).
		},
	}
)

// Sets default regionalized language code.
func SetDefaultLanguageRegion(lang3Letter, region string) {
	lang3Letter = strings.ToLower(lang3Letter)
	lang2Letter := to2LetterCode(lang3Letter)
	region = strings.ToLower(region)
	region = strings.Replace(region, lang2Letter, lang3Letter, 1)

	lang3LetterIndex, ok := langRegionalsMapping[lang3Letter]
	if !ok {
		lang3LetterIndex = map[string]string{}
		langRegionalsMapping[lang3Letter] = lang3LetterIndex
	}

	lang3LetterIndex[""] = region

	lang2LetterIndex, ok := langRegionalsMapping[to2LetterCode(lang3Letter)]
	if !ok {
		lang2LetterIndex = map[string]string{}
		langRegionalsMapping[lang3Letter] = lang2LetterIndex
	}

	lang2LetterIndex[""] = region
}

// Returns regionalized language code. If not found, it returns the original language code.
func ToLanguageRegionalized(language string, override bool) string {
	parts := strings.Split(language, "-")
	lang := strings.ToLower(parts[0])

	var countryCode string
	if len(parts) > 1 {
		countryCode = strings.ToLower(parts[1])
	}

	if override {
		countryCode = ""
	}

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

func ToUpperFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func to2LetterCode(lang3Letter string) string {
	switch lang3Letter {
	case "eng":
		return "en"
	case "fre":
		return "fr"
	case "ger":
		return "de"
	case "spa":
		return "es"
	default:
		return lang3Letter
	}
}
