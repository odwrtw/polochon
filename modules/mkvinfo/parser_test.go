package mkvinfo

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/remko/go-mkvparse"

	polochon "github.com/odwrtw/polochon/lib"
)

// testElement encodes an EBML element, including a variable-length size.
func testElement(id mkvparse.ElementID, data []byte) []byte {
	out := bytes.TrimLeft(binary.BigEndian.AppendUint32(nil, uint32(id)), "\x00")
	width := 1
	for uint64(len(data)) >= (uint64(1)<<(7*width))-1 {
		width++
	}
	size := binary.BigEndian.AppendUint64(nil, uint64(len(data))|uint64(1)<<(7*width))
	out = append(out, size[len(size)-width:]...)
	return append(out, data...)
}

func subtitleTrack(number byte, fields ...[]byte) []byte {
	data := testElement(mkvparse.TrackNumberElement, []byte{number})
	data = append(data, testElement(mkvparse.TrackUIDElement, []byte{number})...)
	data = append(data, testElement(mkvparse.TrackTypeElement, []byte{byte(TrackTypeSubtitle)})...)
	data = append(data, testElement(mkvparse.CodecIDElement, []byte("S_TEXT/UTF8"))...)
	data = append(data, bytes.Join(fields, nil)...)
	return testElement(mkvparse.TrackEntryElement, data)
}

func writeTestMKV(t *testing.T, path string, tracks ...[]byte) {
	t.Helper()
	data := testElement(mkvparse.EBMLElement, testElement(mkvparse.DocTypeElement, []byte("matroska")))
	data = append(data, testElement(mkvparse.SegmentElement, testElement(mkvparse.TracksElement, bytes.Join(tracks, nil)))...)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestParseTrackLanguages(t *testing.T) {
	legacy := func(lang string) []byte { return testElement(mkvparse.LanguageElement, []byte(lang)) }
	ietf := func(lang string) []byte { return testElement(mkvparse.LanguageIETFElement, []byte(lang)) }
	for _, tc := range []struct {
		name   string
		fields [][]byte
		want   polochon.Language
	}{
		{name: "omitted language defaults to English", want: polochon.EN},
		{name: "explicit French", fields: [][]byte{legacy("fre")}, want: polochon.FR},
		{name: "explicit undefined", fields: [][]byte{legacy("und")}},
		{name: "BCP47 only", fields: [][]byte{ietf("fr-FR")}, want: polochon.FR},
		{name: "BCP47 last", fields: [][]byte{legacy("eng"), ietf("fr-FR")}, want: polochon.FR},
		{name: "BCP47 first", fields: [][]byte{ietf("fr-FR"), legacy("eng")}, want: polochon.FR},
		{name: "empty BCP47 overrides default", fields: [][]byte{ietf("")}},
		{name: "unsupported BCP47 overrides default", fields: [][]byte{ietf("de")}},
		{name: "forced flag", fields: [][]byte{testElement(mkvparse.FlagForcedElement, []byte{1})}},
		{name: "forced name", fields: [][]byte{testElement(mkvparse.NameElement, []byte("English Forced"))}},
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

func TestParseTrackLanguageIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "video.mkv")
	writeTestMKV(t, path,
		subtitleTrack(1, testElement(mkvparse.LanguageIETFElement, []byte("fr-FR"))),
		subtitleTrack(2),
		subtitleTrack(3, testElement(mkvparse.LanguageElement, []byte("und"))),
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
