package mkvinfo

import (
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestTrackLanguage(t *testing.T) {
	for _, tc := range []struct {
		name      string
		legacy    string
		ietf      string
		present   bool
		trackName string
		want      polochon.Language
	}{
		{name: "zero value"},
		{name: "legacy English", legacy: "eng", want: polochon.EN},
		{name: "legacy French", legacy: "fre", want: polochon.FR},
		{name: "legacy unknown", legacy: "und"},
		{name: "legacy name fallback", legacy: "und", trackName: "English SDH", want: polochon.EN},
		{name: "missing legacy name fallback", trackName: "US subtitles", want: polochon.EN},
		{name: "English BCP47", ietf: "en", present: true, want: polochon.EN},
		{name: "English regional BCP47", ietf: "en-GB", present: true, want: polochon.EN},
		{name: "French regional BCP47", ietf: "fr-CA", present: true, want: polochon.FR},
		{name: "case insensitive BCP47", ietf: "FR-fr", present: true, want: polochon.FR},
		{name: "three letter BCP47", ietf: "fra", present: true, want: polochon.FR},
		{name: "BCP47 overrides legacy and name", legacy: "eng", ietf: "fr-FR", present: true, trackName: "English", want: polochon.FR},
		{name: "empty BCP47", legacy: "eng", present: true, trackName: "English"},
		{name: "malformed BCP47", legacy: "eng", ietf: "fr-!", present: true, trackName: "English"},
		{name: "unsupported BCP47", legacy: "eng", ietf: "de-DE", present: true, trackName: "English"},
		{name: "undefined BCP47", legacy: "eng", ietf: "und", present: true, trackName: "English"},
		{name: "undefined regional BCP47", legacy: "eng", ietf: "und-US", present: true},
		{name: "private use BCP47", legacy: "eng", ietf: "x-english", present: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			track := &TrackEntry{Language: tc.legacy, Name: tc.trackName}
			if tc.present {
				track.LanguageIETF = &tc.ietf
			}
			got, ok := track.Lang()
			if got != tc.want || ok != (tc.want != "") {
				t.Fatalf("Lang() = %q, %v; want %q, %v", got, ok, tc.want, tc.want != "")
			}
		})
	}
}
