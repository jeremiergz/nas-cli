package sync

import "testing"

func TestSelectReferenceStream(t *testing.T) {
	streams := []probedStream{
		{Index: 0, CodecType: "video"},
		{Index: 1, CodecType: "audio", Tags: map[string]string{"language": "eng"}},
		{
			Index:     2,
			CodecType: "audio",
			Tags:      map[string]string{"language": "fra"},
			Disposition: struct {
				Default int `json:"default"`
			}{Default: 1},
		},
		{Index: 3, CodecType: "subtitle", Tags: map[string]string{"language": "fre"}},
	}

	tests := []struct {
		name     string
		kind     string
		language string
		want     referenceStream
		wantErr  bool
	}{
		{name: "default audio stream", kind: "audio", want: referenceStream{index: 2, language: "fre"}},
		{name: "specified stream language", kind: "audio", language: "eng", want: referenceStream{index: 1, language: "eng"}},
		{name: "French alias", kind: "audio", language: "fre", want: referenceStream{index: 2, language: "fre"}},
		{name: "French IETF alias", kind: "audio", language: "fr", want: referenceStream{index: 2, language: "fre"}},
		{name: "subtitle stream", kind: "subtitle", want: referenceStream{index: 3, language: "fre"}},
		{name: "missing language", kind: "audio", language: "deu", wantErr: true},
		{name: "missing type", kind: "video", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := selectReferenceStream(streams, test.kind, test.language)
			if (err != nil) != test.wantErr {
				t.Fatalf("selectReferenceStream() error = %v, wantErr %v", err, test.wantErr)
			}
			if err == nil && got != test.want {
				t.Errorf("selectReferenceStream() = %+v, want %+v", got, test.want)
			}
		})
	}
}
