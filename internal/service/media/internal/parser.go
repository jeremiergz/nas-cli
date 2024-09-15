package internal

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	patterns = []struct {
		name   string
		isLast bool
		kind   reflect.Kind
		regexp *regexp.Regexp
	}{
		{"season", false, reflect.Int, regexp.MustCompile(`(?i)(s?([0-9]{1,2}))[ex]`)},
		{"episode", false, reflect.Int, regexp.MustCompile(`(?i)([ex]([0-9]{2})(?:[^0-9]|$))`)},
		{"episode", false, reflect.Int, regexp.MustCompile(`(-\s+([0-9]{1,})(?:[^0-9]|$))`)},
		{"year", true, reflect.Int, regexp.MustCompile(`\b(((?:19[0-9]|20[0-9])[0-9]))\b`)},

		{"resolution", false, reflect.String, regexp.MustCompile(`\b(([0-9]{3,4}p))\b`)},
		{"quality", false, reflect.String, regexp.MustCompile(`(?i)\b(((?:PPV\.)?[HP]DTV|(?:HD)?CAM|B[DR]Rip|(?:HD-?)?TS|(?:PPV )?WEB-?DL(?: DVDRip)?|HDRip|DVDRip|DVDRIP|CamRip|W[EB]BRip|BluRay|DvDScr|telesync))\b`)},
		{"codec", false, reflect.String, regexp.MustCompile(`(?i)\b((xvid|[hx]\.?26[45]))\b`)},
		{"audio", false, reflect.String, regexp.MustCompile(`(?i)\b((MP3|DD5\.?1|Dual[\- ]Audio|LiNE|DTS|AAC[.-]LC|AAC(?:\.?2\.0)?|AC3(?:\.5\.1)?))\b`)},
		{"region", false, reflect.String, regexp.MustCompile(`(?i)\b(R([0-9]))\b`)},
		{"size", false, reflect.String, regexp.MustCompile(`(?i)\b((\d+(?:\.\d+)?(?:GB|MB)))\b`)},
		{"website", false, reflect.String, regexp.MustCompile(`^(\[ ?([^\]]+?) ?\])`)},
		{"language", false, reflect.String, regexp.MustCompile(`(?i)\b((rus\.eng|ita\.eng))\b`)},
		{"sbs", false, reflect.String, regexp.MustCompile(`(?i)\b(((?:Half-)?SBS))\b`)},
		{"container", false, reflect.String, regexp.MustCompile(`(?i)\b((MKV|AVI|MP4))\b`)},

		{"group", false, reflect.String, regexp.MustCompile(`\b(- ?([^-]+(?:-={[^-]+-?$)?))$`)},

		{"extended", false, reflect.Bool, regexp.MustCompile(`(?i)\b(EXTENDED(:?.CUT)?)\b`)},
		{"hardcoded", false, reflect.Bool, regexp.MustCompile(`(?i)\b((HC))\b`)},
		{"proper", false, reflect.Bool, regexp.MustCompile(`(?i)\b((PROPER))\b`)},
		{"repack", false, reflect.Bool, regexp.MustCompile(`(?i)\b((REPACK))\b`)},
		{"widescreen", false, reflect.Bool, regexp.MustCompile(`(?i)\b((WS))\b`)},
		{"unrated", false, reflect.Bool, regexp.MustCompile(`(?i)\b((UNRATED))\b`)},
		{"threeD", false, reflect.Bool, regexp.MustCompile(`(?i)\b((3D))\b`)},
	}
)

func init() {
	for _, pattern := range patterns {
		if pattern.regexp.NumSubexp() != 2 {
			panic(fmt.Errorf(
				"pattern %q does not have enough capture groups. Want 2, got %d",
				pattern.name,
				pattern.regexp.NumSubexp(),
			))
		}
	}
}

type DownloadedFile struct {
	Title      string
	Season     int    `json:"season,omitempty"`
	Episode    int    `json:"episode,omitempty"`
	Year       int    `json:"year,omitempty"`
	Resolution string `json:"resolution,omitempty"`
	Quality    string `json:"quality,omitempty"`
	Codec      string `json:"codec,omitempty"`
	Audio      string `json:"audio,omitempty"`
	Group      string `json:"group,omitempty"`
	Region     string `json:"region,omitempty"`
	Extended   bool   `json:"extended,omitempty"`
	Hardcoded  bool   `json:"hardcoded,omitempty"`
	Proper     bool   `json:"proper,omitempty"`
	Repack     bool   `json:"repack,omitempty"`
	Container  string `json:"container,omitempty"`
	Widescreen bool   `json:"widescreen,omitempty"`
	Website    string `json:"website,omitempty"`
	Language   string `json:"language,omitempty"`
	Sbs        string `json:"sbs,omitempty"`
	Unrated    bool   `json:"unrated,omitempty"`
	Size       string `json:"size,omitempty"`
	ThreeD     bool   `json:"3d,omitempty"`
}

func Parse(filename string) (*DownloadedFile, error) {
	file := &DownloadedFile{}

	var startIndex, endIndex = 0, len(filename)
	cleanName := strings.Replace(filename, "_", " ", -1)
	for _, pattern := range patterns {
		matches := pattern.regexp.FindAllStringSubmatch(cleanName, -1)
		if len(matches) == 0 {
			continue
		}

		matchIdx := 0
		if pattern.isLast {
			matchIdx = len(matches) - 1
		}

		index := strings.Index(cleanName, matches[matchIdx][1])
		if index == 0 {
			startIndex = len(matches[matchIdx][1])

		} else if index < endIndex {
			endIndex = index
		}
		setField(file, pattern.name, matches[matchIdx][2])
	}

	raw := strings.Split(filename[startIndex:endIndex], "(")[0]
	cleanName = raw
	if strings.HasPrefix(cleanName, "- ") {
		cleanName = raw[2:]
	}

	if strings.ContainsRune(cleanName, '.') && !strings.ContainsRune(cleanName, ' ') {
		cleanName = strings.Replace(cleanName, ".", " ", -1)
	}

	cleanName = strings.Replace(cleanName, "_", " ", -1)
	setField(file, "title", strings.TrimSpace(cleanName))

	return file, nil
}

var (
	caser = cases.Title(language.Und)
)

func setField(file *DownloadedFile, field, val string) {
	kind := reflect.TypeOf(file)
	value := reflect.ValueOf(file)
	field = caser.String(field)
	v, _ := kind.Elem().FieldByName(field)

	switch v.Type.Kind() {
	case reflect.Bool:
		value.Elem().FieldByName(field).SetBool(true)

	case reflect.Int:
		clean, _ := strconv.ParseInt(val, 10, 64)
		value.Elem().FieldByName(field).SetInt(clean)

	case reflect.Uint:
		clean, _ := strconv.ParseUint(val, 10, 64)
		value.Elem().FieldByName(field).SetUint(clean)

	case reflect.String:
		value.Elem().FieldByName(field).SetString(val)
	}
}
