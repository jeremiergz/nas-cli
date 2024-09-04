package util

import (
	"strings"
	"unicode"

	"github.com/manifoldco/promptui"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	StyleError   = promptui.Styler(promptui.FGRed)
	StyleWarning = promptui.Styler(promptui.FGYellow)
	StyleSuccess = promptui.Styler(promptui.FGGreen)
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

// Removes all diacritics from given string
func RemoveDiacritics(s string) (string, error) {
	output, _, err := transform.String(diacriticsTransformer, s)
	if err != nil {
		return s, err
	}

	return output, nil
}
