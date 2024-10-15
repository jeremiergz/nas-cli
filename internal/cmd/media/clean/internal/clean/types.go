package clean

type attachmentsItems struct {
	ContentType string      `json:"content_type,omitempty"`
	Description string      `json:"description,omitempty"`
	Filename    string      `json:"file_name"`
	ID          int         `json:"id"`
	Properties  *properties `json:"properties"`
	Size        int         `json:"size"`
	Type        string      `json:"type,omitempty"`
}

type chaptersItems struct {
	NumEntries int `json:"num_entries"`
}

type container struct {
	// Additional properties for the container varying by container format.
	Properties *properties `json:"properties,omitempty"`

	// States whether or not mkvmerge knows about the format.
	Recognized bool `json:"recognized"`

	// States whether or not mkvmerge can read the format.
	Supported bool `json:"supported"`

	// A human-readable description/name for the container format.
	Type string `json:"type,omitempty"`
}

type mkvmergeIdentificationOutput struct {
	// An array describing the attachments found if any.
	Attachments []*attachmentsItems `json:"attachments,omitempty"`
	Chapters    []*chaptersItems    `json:"chapters,omitempty"`

	// Information about the identified container.
	Container *container `json:"container,omitempty"`

	// Identified file's name.
	Filename   string        `json:"file_name,omitempty"`
	GlobalTags []interface{} `json:"global_tags,omitempty"`

	// Output format's version.
	IdentificationFormatVersion int               `json:"identification_format_version,omitempty"`
	TrackTags                   []*trackTagsItems `json:"track_tags,omitempty"`
	Tracks                      []*tracksItems    `json:"tracks,omitempty"`
}

type properties struct {
	AacIsSbr                  string `json:"aac_is_sbr,omitempty"`
	AudioBitsPerSample        int    `json:"audio_bits_per_sample,omitempty"`
	AudioChannels             int    `json:"audio_channels,omitempty"`
	AudioSamplingFrequency    int    `json:"audio_sampling_frequency,omitempty"`
	CodecID                   string `json:"codec_id,omitempty"`
	CodecPrivateData          string `json:"codec_private_data,omitempty"`
	CodecPrivateLength        int    `json:"codec_private_length,omitempty"`
	ContentEncodingAlgorithms string `json:"content_encoding_algorithms,omitempty"`
	DefaultDuration           int    `json:"default_duration,omitempty"`
	DefaultTrack              bool   `json:"default_track,omitempty"`
	DisplayDimensions         string `json:"display_dimensions,omitempty"`
	EnabledTrack              bool   `json:"enabled_track,omitempty"`
	ForcedTrack               bool   `json:"forced_track,omitempty"`
	Language                  string `json:"language,omitempty"`
	Number                    int    `json:"number,omitempty"`
	Packetizer                string `json:"packetizer,omitempty"`
	PixelDimensions           string `json:"pixel_dimensions,omitempty"`
	StereoMode                int    `json:"stereo_mode,omitempty"`
	StreamID                  string `json:"stream_id,omitempty"`
	SubStreamID               string `json:"sub_stream_id,omitempty"`
	TagArtist                 string `json:"tag_artist,omitempty"`
	TagBitsps                 string `json:"tag_bitsps,omitempty"`
	TagBps                    string `json:"tag_bps,omitempty"`
	TagFps                    string `json:"tag_fps,omitempty"`
	TagTitle                  string `json:"tag_title,omitempty"`
	TextSubtitles             bool   `json:"text_subtitles,omitempty"`
	TrackName                 string `json:"track_name,omitempty"`
	TsPid                     int    `json:"ts_pid,omitempty"`
}

type trackTagsItems struct {
	NumEntries int `json:"num_entries"`
	TrackID    int `json:"track_id"`
}

type tracksItems struct {
	Codec      string      `json:"codec"`
	ID         int         `json:"id"`
	Properties *properties `json:"properties,omitempty"`
	Type       string      `json:"type"`
}
