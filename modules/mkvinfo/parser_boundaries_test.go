package mkvinfo

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"testing/iotest"
)

func TestParseTracksSegmentSizes(t *testing.T) {
	tracks := testElement(tracksElement, subtitleTrack(1))
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{name: "known size", data: testElement(segmentElement, tracks)},
		{name: "unknown size", data: append([]byte{0x18, 0x53, 0x80, 0x67, 0xFF}, tracks...)},
		{name: "Tracks after Cluster", data: testElement(segmentElement, append(testElement(0x1F43B675, testElement(0xE7, []byte{0})), tracks...))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entries, err := parseTracks(bytes.NewReader(tc.data))
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || entries[0].Codec != "S_TEXT/UTF8" {
				t.Fatalf("unexpected tracks: %+v", entries)
			}
		})
	}
}

func TestParseTracksStopsBeforeMedia(t *testing.T) {
	// A malformed unknown-size Cluster must not be visited after Tracks.
	tail := []byte{0x1F, 0x43, 0xB6, 0x75, 0xFF, 0, 0, 0}
	data := testElement(segmentElement, append(testElement(tracksElement, subtitleTrack(1)), tail...))
	r := bytes.NewReader(data)
	entries, err := parseTracks(r)
	if err != nil || len(entries) != 1 {
		t.Fatalf("parseTracks() = %+v, %v", entries, err)
	}
	if r.Len() != len(tail) {
		t.Fatalf("%d unread bytes, want %d", r.Len(), len(tail))
	}
}

func TestParseTracksEOFWithData(t *testing.T) {
	complete := testElement(segmentElement, testElement(tracksElement, subtitleTrack(1)))
	truncated := testElement(segmentElement, testElement(tracksElement, append(subtitleTrack(1), 0)))
	for _, tc := range []struct {
		name string
		data []byte
		want error
	}{
		{name: "complete", data: complete},
		{name: "truncated", data: truncated[:len(truncated)-1], want: io.ErrUnexpectedEOF},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entries, err := parseTracks(iotest.DataErrReader(bytes.NewReader(tc.data)))
			if !errors.Is(err, tc.want) {
				t.Fatalf("parseTracks() error = %v, want %v", err, tc.want)
			}
			if tc.want == nil && len(entries) != 1 {
				t.Fatalf("got %d tracks, want 1", len(entries))
			}
			if tc.want != nil && entries != nil {
				t.Fatal("truncated file returned partial tracks")
			}
		})
	}
}

func TestParseTracksEmpty(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{name: "no Tracks", data: testElement(segmentElement, nil)},
		{name: "empty Tracks", data: testElement(segmentElement, testElement(tracksElement, nil))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entries, err := parseTracks(bytes.NewReader(tc.data))
			if err != nil || entries != nil {
				t.Fatalf("parseTracks() = %+v, %v; want nil, nil", entries, err)
			}
		})
	}
}

func TestParseTracksMalformed(t *testing.T) {
	scalar := testElement(segmentElement, testElement(tracksElement, subtitleTrack(1)))
	trackBoundary := testElement(segmentElement, testElement(tracksElement, subtitleTrack(1, []byte{0})))
	tracksBoundary := testElement(segmentElement, testElement(tracksElement, append(subtitleTrack(1), 0)))
	for _, tc := range []struct {
		name string
		data []byte
		want error
	}{
		{name: "truncated header", data: []byte{0x18, 0x53, 0x80, 0x67}, want: io.ErrUnexpectedEOF},
		{name: "truncated scalar", data: scalar[:len(scalar)-1], want: io.ErrUnexpectedEOF},
		{name: "truncated TrackEntry at boundary", data: trackBoundary[:len(trackBoundary)-1], want: io.ErrUnexpectedEOF},
		{name: "truncated Tracks at boundary", data: tracksBoundary[:len(tracksBoundary)-1], want: io.ErrUnexpectedEOF},
		{name: "oversized integer", data: testElement(segmentElement, testElement(tracksElement, subtitleTrack(1, testElement(flagForcedElement, make([]byte, 9))))), want: ErrInvalidElementSize},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entries, err := parseTracks(bytes.NewReader(tc.data))
			if !errors.Is(err, tc.want) || entries != nil {
				t.Fatalf("parseTracks() = %+v, %v; want nil, %v", entries, err, tc.want)
			}
		})
	}
}
