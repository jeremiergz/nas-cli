package util

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type Alphabetic []string

func (list Alphabetic) Len() int { return len(list) }

func (list Alphabetic) Swap(i, j int) { list[i], list[j] = list[j], list[i] }

func (list Alphabetic) Less(i, j int) bool {
	return (strings.ToLower(list[i]))[0] < (strings.ToLower(list[j]))[0]
}

var diacriticsTransformer = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

// Removes all diacritics from given string
func RemoveDiacritics(s string) (string, error) {
	output, _, err := transform.String(diacriticsTransformer, s)
	if err != nil {
		return s, err
	}

	return output, nil
}

// Checks whether given string is in given array or not
func StringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}

	return false
}
