package util

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type SortAlphabetic []string

func (list SortAlphabetic) Len() int { return len(list) }

func (list SortAlphabetic) Less(i, j int) bool {
	return (strings.ToLower(list[i])) < (strings.ToLower(list[j]))
}

func (list SortAlphabetic) Swap(i, j int) { list[i], list[j] = list[j], list[i] }

var (
	diacriticsTransformer = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
)

// Checks whether given generic value is in array
func Contains[T comparable](elems []T, v T) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}

	return false
}

// Removes all diacritics from given string
func RemoveDiacritics(s string) (string, error) {
	output, _, err := transform.String(diacriticsTransformer, s)
	if err != nil {
		return s, err
	}

	return output, nil
}
