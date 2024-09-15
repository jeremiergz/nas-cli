package mkv

// AttachmentsItems
type AttachmentsItems struct {
	ContentType string      `json:"content_type,omitempty"`
	Description string      `json:"description,omitempty"`
	Filename    string      `json:"file_name"`
	ID          int         `json:"id"`
	Properties  *Properties `json:"properties"`
	Size        int         `json:"size"`
	Type        string      `json:"type,omitempty"`
}

// ChaptersItems
type ChaptersItems struct {
	NumEntries int `json:"num_entries"`
}

// Container information about the identified container
type Container struct {

	// additional properties for the container varying by container format
	Properties *Properties `json:"properties,omitempty"`

	// States whether or not mkvmerge knows about the format
	Recognized bool `json:"recognized"`

	// States whether or not mkvmerge can read the format
	Supported bool `json:"supported"`

	// A human-readable description/name for the container format
	Type string `json:"type,omitempty"`
}

// MkvmergeIdentificationOutput The JSON output produced by mkvmerge's file identification mode
type MkvmergeIdentificationOutput struct {

	// an array describing the attachments found if any
	Attachments []*AttachmentsItems `json:"attachments,omitempty"`
	Chapters    []*ChaptersItems    `json:"chapters,omitempty"`

	// information about the identified container
	Container *Container `json:"container,omitempty"`

	// the identified file's name
	Filename   string        `json:"file_name,omitempty"`
	GlobalTags []interface{} `json:"global_tags,omitempty"`

	// The output format's version
	IdentificationFormatVersion int               `json:"identification_format_version,omitempty"`
	TrackTags                   []*TrackTagsItems `json:"track_tags,omitempty"`
	Tracks                      []*TracksItems    `json:"tracks,omitempty"`
}

// Properties
type Properties struct {
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

// TrackTagsItems
type TrackTagsItems struct {
	NumEntries int `json:"num_entries"`
	TrackID    int `json:"track_id"`
}

// TracksItems
type TracksItems struct {
	Codec      string      `json:"codec"`
	ID         int         `json:"id"`
	Properties *Properties `json:"properties,omitempty"`
	Type       string      `json:"type"`
}
