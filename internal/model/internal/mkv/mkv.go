package mkv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

// Cleans given file tracks.
func CleanTracks(filePath, basename string) error {
	characteristics, err := getCharacteristics(filePath)
	if err != nil {
		return err
	}

	options := []string{
		"--edit",
		"info",
		"--set",
		"title=",
		"--tags",
		"all:",
	}

	audioTrackNumber := 1
	subtitleTrackNumber := 1
	videoTrackNumber := 1

	for _, track := range characteristics.Tracks {
		switch track.Type {
		case "audio":
			options = append(options,
				"--edit",
				fmt.Sprintf("track:a%d", audioTrackNumber),
				"--set",
				fmt.Sprintf("language=%s", util.ToLanguageRegionalized(track.Properties.Language)),
				"--set",
				"name=",
			)
			audioTrackNumber++

		case "subtitles":
			isForced := false
			normalizedTrackName := strings.ToLower(track.Properties.TrackName)
			if track.Properties.ForcedTrack || strings.Contains(normalizedTrackName, "forc") {
				isForced = true
			}

			options = append(options,
				"--edit",
				fmt.Sprintf("track:s%d", subtitleTrackNumber),
				"--set",
				fmt.Sprintf("language=%s", util.ToLanguageRegionalized(track.Properties.Language)),
				"--set",
				fmt.Sprintf("name=%s", util.ToLanguageDisplayName(track.Properties.Language, isForced)),
			)
			if isForced {
				options = append(options,
					"--set",
					"flag-default=1",
					"--set",
					"flag-forced=1",
				)
			} else {
				options = append(options,
					"--set",
					"flag-default=0",
					"--set",
					"flag-forced=0",
				)
			}
			subtitleTrackNumber++

		case "video":
			options = append(options,
				"--edit",
				fmt.Sprintf("track:v%d", videoTrackNumber),
				"--set",
				"language=und",
				"--set",
				"name=",
			)
			videoTrackNumber++
		}
	}

	options = append(options, filePath)

	propedit := exec.Command(cmdutil.CommandMKVPropEdit, options...)

	err = propedit.Run()
	if err != nil {
		return err
	}

	return nil
}

// Retrieves the characteristics of given MKV file.
func getCharacteristics(filePath string) (*MkvmergeIdentificationOutput, error) {
	options := []string{
		"--identification-format",
		"json",
		"--identify",
		filePath,
	}

	buf := new(bytes.Buffer)

	merge := exec.Command(cmdutil.CommandMKVMerge, options...)
	merge.Stdout = buf
	merge.Stderr = os.Stderr
	err := merge.Run()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve characteristics: %w", err)
	}

	var characteristics *MkvmergeIdentificationOutput
	err = json.Unmarshal(buf.Bytes(), &characteristics)
	if err != nil {
		return nil, fmt.Errorf("unable to parse characteristics: %w", err)
	}

	return characteristics, nil
}
