package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/jeremiergz/nas-cli/internal/util/cmdutil"
)

type probedStream struct {
	Index       int               `json:"index"`
	CodecType   string            `json:"codec_type"`
	Tags        map[string]string `json:"tags"`
	Disposition struct {
		Default int `json:"default"`
	} `json:"disposition"`
}

type probedStreams struct {
	Streams []probedStream `json:"streams"`
}

type referenceStream struct {
	index    int
	language string
}

func probeReferenceStream(ctx context.Context, videoPath, streamType, language string) (referenceStream, error) {
	args := []string{
		"-v", "error",
		"-show_entries", "stream=index,codec_type:stream_disposition=default:stream_tags=language,LANGUAGE,language_ietf",
		"-of", "json",
		videoPath,
	}
	output, err := exec.CommandContext(ctx, cmdutil.CommandFFprobe, args...).Output()
	if err != nil {
		return referenceStream{}, fmt.Errorf("ffprobe failed: %w", err)
	}

	var result probedStreams
	if err := json.Unmarshal(output, &result); err != nil {
		return referenceStream{}, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}
	return selectReferenceStream(result.Streams, streamType, language)
}

func selectReferenceStream(streams []probedStream, streamType, language string) (referenceStream, error) {
	streamType = strings.ToLower(strings.TrimSpace(streamType))
	if streamType == "sub" {
		streamType = "subtitle"
	}
	if streamType != "audio" && streamType != "subtitle" {
		return referenceStream{}, fmt.Errorf("unsupported reference stream type %q", streamType)
	}

	var candidates []probedStream
	for _, stream := range streams {
		if stream.CodecType == streamType {
			candidates = append(candidates, stream)
		}
	}
	if len(candidates) == 0 {
		return referenceStream{}, fmt.Errorf("no %s stream found", streamType)
	}

	if language != "" {
		normalizedLanguage := normalizeSubsyncLanguage(language)
		for _, candidate := range candidates {
			candidateLanguage := normalizeSubsyncLanguage(streamLanguage(candidate))
			if candidateLanguage == normalizedLanguage {
				return referenceStream{index: candidate.Index, language: candidateLanguage}, nil
			}
		}
		return referenceStream{}, fmt.Errorf("no %s stream found with language %q", streamType, language)
	}

	for _, candidate := range candidates {
		if candidate.Disposition.Default != 0 {
			return referenceStream{
				index:    candidate.Index,
				language: normalizeSubsyncLanguage(streamLanguage(candidate)),
			}, nil
		}
	}

	return referenceStream{
		index:    candidates[0].Index,
		language: normalizeSubsyncLanguage(streamLanguage(candidates[0])),
	}, nil
}

func streamLanguage(stream probedStream) string {
	for _, key := range []string{"language", "language_ietf"} {
		for tag, language := range stream.Tags {
			if !strings.EqualFold(tag, key) {
				continue
			}
			language = strings.ToLower(strings.TrimSpace(language))
			if primary, _, ok := strings.Cut(language, "-"); ok {
				language = primary
			}
			if language != "" && language != "und" {
				return language
			}
		}
	}
	return ""
}
