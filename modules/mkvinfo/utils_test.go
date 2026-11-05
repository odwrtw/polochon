package mkvinfo

import (
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestMetadataCodecs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		kind   TrackType
		codecs []string
		want   string
	}{
		{name: "audio recognized then unknown", kind: TrackTypeAudio, codecs: []string{"A_AC3", "A_DTS"}, want: "Dolby Digital"},
		{name: "audio unknown then recognized", kind: TrackTypeAudio, codecs: []string{"A_DTS", "A_AAC"}, want: "AAC"},
		{name: "audio last recognized wins", kind: TrackTypeAudio, codecs: []string{"A_AAC", "A_EAC3"}, want: "Dolby Digital Plus"},
		{name: "audio unknown only", kind: TrackTypeAudio, codecs: []string{"A_DTS"}},
		{name: "video recognized then unknown", kind: TrackTypeVideo, codecs: []string{"V_MPEG4/ISO/AVC", "V_AV1"}, want: "H.264"},
		{name: "video unknown then recognized", kind: TrackTypeVideo, codecs: []string{"V_AV1", "V_MPEGH/ISO/HEVC"}, want: "H.265"},
		{name: "video last recognized wins", kind: TrackTypeVideo, codecs: []string{"V_MPEG4/ISO/AVC", "V_MPEGH/ISO/HEVC"}, want: "H.265"},
		{name: "video unknown only", kind: TrackTypeVideo, codecs: []string{"V_AV1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var tracks []*TrackEntry
			for _, codec := range tc.codecs {
				tracks = append(tracks, &TrackEntry{Type: tc.kind, Codec: codec})
			}
			got := Metadata(tracks)
			if tc.want == "" {
				if got != nil {
					t.Fatalf("Metadata() = %+v, want nil", got)
				}
				return
			}
			want := &polochon.VideoMetadata{Container: "mkv"}
			if tc.kind == TrackTypeAudio {
				want.AudioCodec = tc.want
			} else {
				want.VideoCodec = tc.want
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("Metadata() = %+v, want %+v", got, want)
			}
		})
	}
}

func TestMetadataMixedTracks(t *testing.T) {
	got := Metadata([]*TrackEntry{
		{Type: TrackTypeAudio, Codec: "A_AC3"},
		{Type: TrackTypeVideo, Codec: "V_MPEG4/ISO/AVC"},
		{Type: TrackTypeSubtitle, Language: "fre"},
		{Type: TrackTypeSubtitle, Language: "fre"},
		{Type: TrackTypeSubtitle, Language: "eng", Forced: true},
		{Type: TrackTypeAudio, Codec: "A_DTS"},
		{Type: TrackTypeVideo, Codec: "V_AV1"},
	})
	want := &polochon.VideoMetadata{
		Container: "mkv", AudioCodec: "Dolby Digital", VideoCodec: "H.264",
		EmbeddedSubtitles: []polochon.Language{polochon.FR},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Metadata() = %+v, want %+v", got, want)
	}
	if got := Metadata(nil); got != nil {
		t.Fatalf("Metadata(nil) = %+v, want nil", got)
	}
}
