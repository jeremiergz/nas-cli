package sync

import "testing"

func TestLanguageForSubtitle(t *testing.T) {
	tests := []struct {
		name     string
		subtitle string
		override string
		want     string
	}{
		{name: "infer French", subtitle: "Movie.fra.srt", want: "fre"},
		{name: "infer legacy French", subtitle: "Movie.fre.srt", want: "fre"},
		{name: "infer forced subtitle", subtitle: "Movie.fra.forced.srt", want: "fre"},
		{name: "ignore non-language suffix", subtitle: "Movie.1080p.srt", want: ""},
		{name: "fallback without suffix", subtitle: "Movie.srt", want: ""},
		{name: "explicit override", subtitle: "Movie.fra.srt", override: "eng", want: "eng"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := languageForSubtitle(test.subtitle, test.override); got != test.want {
				t.Errorf("languageForSubtitle(%q, %q) = %q, want %q", test.subtitle, test.override, got, test.want)
			}
		})
	}
}

func TestOutputSubtitleName(t *testing.T) {
	tests := []struct {
		video    string
		language string
		want     string
	}{
		{video: "Movie.mkv", language: "fre", want: "Movie.fre.srt"},
		{video: "Movie.mkv", want: "Movie.srt"},
	}
	for _, test := range tests {
		if got := outputSubtitleName(test.video, test.language); got != test.want {
			t.Errorf("outputSubtitleName(%q, %q) = %q, want %q", test.video, test.language, got, test.want)
		}
	}
}
