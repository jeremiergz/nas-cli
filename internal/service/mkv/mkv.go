package mkv

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jeremiergz/nas-cli/internal/model"
	"github.com/jeremiergz/nas-cli/internal/util"
	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

type Service struct{}

func New() *Service {
	return &Service{}
}

type MKVCharacteristics struct {
	Attachments []any         `json:"attachments"`
	Chapters    []any         `json:"chapters"`
	Container   *MKVContainer `json:"container"`
}

type MKVContainer struct {
	Properties *MKVContainerProperties `json:"properties"`
	Recognized bool                    `json:"recognized"`
	Supported  bool                    `json:"supported"`
	Type       string                  `json:"type"`
}

type MKVContainerProperties struct {
	ContainerType         int       `json:"container_type"`
	DateLocal             time.Time `json:"date_local"`
	DateUTC               time.Time `json:"date_utc"`
	Duration              int       `json:"duration"`
	IsProvidingTimestamps bool      `json:"is_providing_timestamps"`
	MuxingApplication     string    `json:"muxing_application"`
	SegmentUID            string    `json:"segment_uid"`
	WritingApplication    string    `json:"writing_application"`
}

// Retrieves the characteristics of given MKV file.
func (s *Service) GetCharacteristics(filePath string) (*MkvmergeIdentificationOutput, error) {
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

// Cleans given TV show episode tracks.
func (s *Service) CleanEpisodeTracks(wd string, ep *model.Episode) util.Result {
	start := time.Now()

	characteristics, err := s.GetCharacteristics(ep.FilePath)
	if err != nil {
		return util.Result{
			Characteristics: map[string]string{
				"duration": time.Since(start).Round(time.Millisecond).String(),
			},
			IsSuccessful: false,
			Message:      ep.Basename,
		}
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

	options = append(options, ep.FilePath)

	propedit := exec.Command(cmdutil.CommandMKVPropEdit, options...)

	err = propedit.Run()
	if err != nil {
		return util.Result{
			Characteristics: map[string]string{
				"duration": time.Since(start).Round(time.Millisecond).String(),
			},
			IsSuccessful: false,
			Message:      ep.Basename,
		}
	}

	return util.Result{
		Characteristics: map[string]string{
			"duration": time.Since(start).Round(time.Second).String(),
		},
		IsSuccessful: true,
		Message:      ep.Basename,
	}
}
