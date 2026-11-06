package mkvinfo

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

// Fixture IDs are independent of the production reader's constants.
const (
	ebmlElement                uint32 = 0x1A45DFA3
	docTypeElement             uint32 = 0x4282
	segmentElement             uint32 = 0x18538067
	tracksElement              uint32 = 0x1654AE6B
	trackEntryElement          uint32 = 0xAE
	trackNumberElement         uint32 = 0xD7
	trackUIDElement            uint32 = 0x73C5
	trackTypeElement           uint32 = 0x83
	codecIDElement             uint32 = 0x86
	languageElement            uint32 = 0x22B59C
	languageIETFElement        uint32 = 0x22B59D
	nameElement                uint32 = 0x536E
	flagForcedElement          uint32 = 0x55AA
	flagHearingImpairedElement uint32 = 0x55AB
	flagOriginalElement        uint32 = 0x55AE
)

// testElement encodes an EBML element, including a variable-length size.
func testElement(id uint32, data []byte) []byte {
	out := bytes.TrimLeft(binary.BigEndian.AppendUint32(nil, id), "\x00")
	width := 1
	for uint64(len(data)) >= (uint64(1)<<(7*width))-1 {
		width++
	}
	size := binary.BigEndian.AppendUint64(nil, uint64(len(data))|uint64(1)<<(7*width))
	out = append(out, size[len(size)-width:]...)
	return append(out, data...)
}

func subtitleTrack(number byte, fields ...[]byte) []byte {
	return testTrack(number, TrackTypeSubtitle, "S_TEXT/UTF8", fields...)
}

func testTrack(number byte, kind TrackType, codec string, fields ...[]byte) []byte {
	data := testElement(trackNumberElement, []byte{number})
	data = append(data, testElement(trackUIDElement, []byte{number})...)
	data = append(data, testElement(trackTypeElement, []byte{byte(kind)})...)
	data = append(data, testElement(codecIDElement, []byte(codec))...)
	data = append(data, bytes.Join(fields, nil)...)
	return testElement(trackEntryElement, data)
}

func writeTestMKV(t *testing.T, path string, tracks ...[]byte) {
	t.Helper()
	data := testElement(ebmlElement, testElement(docTypeElement, []byte("matroska")))
	data = append(data, testElement(segmentElement, testElement(tracksElement, bytes.Join(tracks, nil)))...)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestParseTrackLanguages(t *testing.T) {
	legacy := func(lang string) []byte { return testElement(languageElement, []byte(lang)) }
	ietf := func(lang string) []byte { return testElement(languageIETFElement, []byte(lang)) }
	for _, tc := range []struct {
		name   string
		fields [][]byte
		want   polochon.Language
	}{
		{name: "omitted language defaults to English", want: polochon.EN},
		{name: "explicit French", fields: [][]byte{legacy("fre")}, want: polochon.FR},
		{name: "explicit undefined", fields: [][]byte{legacy("und")}},
		{name: "explicit empty legacy", fields: [][]byte{legacy("")}},
		{name: "BCP47 only", fields: [][]byte{ietf("fr-FR")}, want: polochon.FR},
		{name: "BCP47 last", fields: [][]byte{legacy("eng"), ietf("fr-FR")}, want: polochon.FR},
		{name: "BCP47 first", fields: [][]byte{ietf("fr-FR"), legacy("eng")}, want: polochon.FR},
		{name: "empty BCP47 overrides default", fields: [][]byte{ietf("")}},
		{name: "unsupported BCP47 overrides default", fields: [][]byte{ietf("de")}},
		{name: "forced flag", fields: [][]byte{testElement(flagForcedElement, []byte{1})}},
		{name: "forced name", fields: [][]byte{testElement(nameElement, []byte("English Forced"))}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "video.mkv")
			writeTestMKV(t, path, subtitleTrack(1, tc.fields...))
			file := &polochon.File{Path: path}
			tracks, err := ParseFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if len(tracks) != 1 {
				t.Fatalf("got %d tracks, want 1", len(tracks))
			}
			metadata, err := (&MKVInfo{}).GuessMetadata(file)
			if err != nil {
				t.Fatal(err)
			}
			for _, lang := range []polochon.Language{polochon.EN, polochon.FR} {
				want := lang == tc.want
				if got := HasSubtitle(tracks, lang); got != want {
					t.Errorf("HasSubtitle(%q) = %v, want %v", lang, got, want)
				}
				if got := metadata != nil && slices.Contains(metadata.EmbeddedSubtitles, lang); got != want {
					t.Errorf("Metadata contains %q = %v, want %v", lang, got, want)
				}
			}
		})
	}
}

func TestParseTrackMetadata(t *testing.T) {
	for _, tc := range []struct {
		name       string
		videoCodec string
		audioCodec string
		wantVideo  string
		wantAudio  string
	}{
		{name: "AVC and EAC3", videoCodec: "V_MPEG4/ISO/AVC", audioCodec: "A_EAC3", wantVideo: "H.264", wantAudio: "Dolby Digital Plus"},
		{name: "HEVC and AC3", videoCodec: "V_MPEGH/ISO/HEVC", audioCodec: "A_AC3", wantVideo: "H.265", wantAudio: "Dolby Digital"},
		{name: "unknown video and AAC", videoCodec: "V_UNKNOWN", audioCodec: "A_AAC", wantAudio: "AAC"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "video.mkv")
			writeTestMKV(t, path,
				testTrack(1, TrackTypeVideo, tc.videoCodec,
					// Legacy fields and nested Video metadata are opaque.
					testElement(0x3A9697, []byte("legacy settings")),
					testElement(0xE0, testElement(0x2383E3, []byte{0x41, 0xC0, 0, 0})),
				),
				testTrack(2, TrackTypeAudio, tc.audioCodec),
				subtitleTrack(3, testElement(languageIETFElement, []byte("fr-FR"))),
				subtitleTrack(4, testElement(flagForcedElement, []byte{1})),
			)
			metadata, err := (&MKVInfo{}).GuessMetadata(&polochon.File{Path: path})
			if err != nil {
				t.Fatal(err)
			}
			if metadata == nil || metadata.Container != "mkv" || metadata.VideoCodec != tc.wantVideo || metadata.AudioCodec != tc.wantAudio {
				t.Fatalf("unexpected metadata: %+v", metadata)
			}
			if len(metadata.EmbeddedSubtitles) != 1 || metadata.EmbeddedSubtitles[0] != polochon.FR {
				t.Fatalf("unexpected embedded subtitles: %v", metadata.EmbeddedSubtitles)
			}
		})
	}
}

func TestParseTrackScalarValues(t *testing.T) {
	data := testElement(segmentElement, testElement(tracksElement, subtitleTrack(1,
		testElement(nameElement, []byte("French\x00padding")),
		testElement(languageElement, []byte("eng")),
		testElement(languageElement, []byte("fre\x00")),
		testElement(languageIETFElement, []byte("en-US")),
		testElement(languageIETFElement, []byte("fr-FR\x00")),
		testElement(flagForcedElement, make([]byte, 8)),
	)))
	tracks, err := parseTracks(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 || tracks[0].Name != "French" || tracks[0].Language != "fre" || tracks[0].LanguageIETF == nil || *tracks[0].LanguageIETF != "fr-FR" || tracks[0].Forced {
		t.Fatalf("unexpected tracks: %+v", tracks)
	}
}

func TestParseModernTrackFlags(t *testing.T) {
	for _, value := range []byte{0, 1} {
		t.Run(string('0'+value), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "video.mkv")
			writeTestMKV(t, path,
				subtitleTrack(1,
					testElement(languageIETFElement, []byte("fr-FR")),
					testElement(flagHearingImpairedElement, []byte{value}),
					testElement(flagOriginalElement, []byte{value}),
					testElement(nameElement, []byte("French SDH")),
				),
				subtitleTrack(2),
			)
			tracks, err := ParseFile(&polochon.File{Path: path})
			if err != nil {
				t.Fatal(err)
			}
			if len(tracks) != 2 {
				t.Fatalf("got %d tracks, want 2", len(tracks))
			}
			if tracks[0].Name != "French SDH" || tracks[0].Codec != "S_TEXT/UTF8" || tracks[0].Forced || tracks[1].Forced {
				t.Fatalf("unexpected tracks: %+v, %+v", tracks[0], tracks[1])
			}
			if !HasSubtitle(tracks, polochon.FR) || !HasSubtitle(tracks, polochon.EN) {
				t.Fatal("modern flags changed subtitle language availability")
			}
		})
	}
}

func TestParseTrackLanguageIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "video.mkv")
	writeTestMKV(t, path,
		subtitleTrack(1, testElement(languageIETFElement, []byte("fr-FR"))),
		subtitleTrack(2),
		subtitleTrack(3, testElement(languageElement, []byte("und"))),
	)
	tracks, err := ParseFile(&polochon.File{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 3 {
		t.Fatalf("got %d tracks, want 3", len(tracks))
	}
	for i, want := range []polochon.Language{polochon.FR, polochon.EN, ""} {
		got, ok := tracks[i].Lang()
		if got != want || ok != (want != "") {
			t.Errorf("track %d: Lang() = %q, %v; want %q, %v", i, got, ok, want, want != "")
		}
	}
}
