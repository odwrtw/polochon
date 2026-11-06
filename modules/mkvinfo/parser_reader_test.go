package mkvinfo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
	"testing/iotest"
)

func TestReadVINT(t *testing.T) {
	for width := 1; width <= 8; width++ {
		for _, value := range []uint64{0, 1, uint64(1)<<(7*width) - 1} {
			encoded := binary.BigEndian.AppendUint64(nil, value|uint64(1)<<(7*width))
			got, n, err := readVINT(iotest.OneByteReader(bytes.NewReader(encoded[8-width:])), 8)
			if err != nil || got != value || n != width {
				t.Fatalf("width %d, value %d: readVINT() = %d, %d, %v", width, value, got, n, err)
			}
		}
	}
}

func TestParseTracksSkipsOpaqueElements(t *testing.T) {
	// Unknown and unneeded payloads must not be interpreted, even if they
	// contain malformed EBML or look like additional tracks.
	fakeTrack := subtitleTrack(99)
	track := subtitleTrack(1,
		testElement(0x4ABC, fakeTrack),
		testElement(0x3A9697, []byte{0, 0xFF}), // CodecSettings
		testElement(0xE0, []byte{0, 0xFF}),     // Video
	)
	tracks := testElement(tracksElement, append(testElement(0x4ABC, fakeTrack), track...))
	segment := testElement(segmentElement, append(testElement(0x1941A469, fakeTrack), tracks...))
	data := append(testElement(0x1A45DFA3, []byte{0, 0xFF}), segment...)
	entries, err := parseTracks(bytes.NewReader(data))
	if err != nil || len(entries) != 1 || entries[0].Type != TrackTypeSubtitle {
		t.Fatalf("parseTracks() = %+v, %v", entries, err)
	}
}

func TestParseTracksUnknownCluster(t *testing.T) {
	cluster := []byte{0x1F, 0x43, 0xB6, 0x75, 0xFF}
	// Block contents are opaque, including bytes that look like Tracks.
	cluster = append(cluster, testElement(0xA3, testElement(tracksElement, subtitleTrack(99)))...)
	cluster = append(cluster, 0x1F, 0x43, 0xB6, 0x75, 0xFF) // Another unknown Cluster.
	payload := append(cluster, testElement(tracksElement, subtitleTrack(1))...)
	for _, data := range [][]byte{
		testElement(segmentElement, payload),
		append([]byte{0x18, 0x53, 0x80, 0x67, 0xFF}, payload...),
	} {
		entries, err := parseTracks(bytes.NewReader(data))
		if err != nil || len(entries) != 1 {
			t.Fatalf("parseTracks() = %+v, %v", entries, err)
		}
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func TestParseTracksLargeSkippedPayload(t *testing.T) {
	// A 128 MiB + 1 byte attachment before Tracks must not be buffered or
	// subject to the string limit. Generate its contents without allocating it.
	const size = 128*1024*1024 + 1
	for _, id := range []uint32{0xEC, 0x1941A469} { // Void and Attachments.
		header := bytes.TrimLeft(binary.BigEndian.AppendUint32(nil, id), "\x00")
		header = binary.BigEndian.AppendUint32(header, uint32(size)|0x10000000)
		r := io.MultiReader(
			bytes.NewReader([]byte{0x18, 0x53, 0x80, 0x67, 0xFF}),
			bytes.NewReader(header),
			io.LimitReader(zeroReader{}, size),
			bytes.NewReader(testElement(tracksElement, subtitleTrack(1))),
		)
		entries, err := parseTracks(r)
		if err != nil || len(entries) != 1 {
			t.Fatalf("element %x: parseTracks() = %+v, %v", id, entries, err)
		}
	}
}

func TestParseTracksRejectsMalformedElements(t *testing.T) {
	wrap := func(field []byte) []byte {
		return testElement(segmentElement, testElement(tracksElement, subtitleTrack(1, field)))
	}
	for _, tc := range []struct {
		name string
		data []byte
		want error
	}{
		{name: "zero VINT", data: []byte{0}, want: ErrInvalidElement},
		{name: "reserved ID", data: []byte{0xFF, 0x80}, want: ErrInvalidElement},
		{name: "long ID", data: []byte{0x08, 0, 0, 0, 0, 0x80}, want: ErrInvalidElement},
		{name: "zero size VINT", data: []byte{0xEC, 0}, want: ErrInvalidElement},
		{name: "partial ID", data: []byte{0x1A, 0x45}, want: io.ErrUnexpectedEOF},
		{name: "partial size", data: []byte{0xEC, 0x40}, want: io.ErrUnexpectedEOF},
		{name: "truncated skipped payload", data: []byte{0xEC, 0x81}, want: io.ErrUnexpectedEOF},
		{name: "truncated Segment without Tracks", data: []byte{0x18, 0x53, 0x80, 0x67, 0x81}, want: io.ErrUnexpectedEOF},
		{name: "unknown-size skipped payload", data: []byte{0xEC, 0xFF}, want: ErrInvalidElementSize},
		{name: "unknown-size Tracks", data: testElement(segmentElement, []byte{0x16, 0x54, 0xAE, 0x6B, 0xFF}), want: ErrInvalidElementSize},
		{name: "unknown-size TrackEntry", data: testElement(segmentElement, testElement(tracksElement, []byte{0xAE, 0xFF})), want: ErrInvalidElementSize},
		{name: "unknown-size field", data: wrap([]byte{0x4A, 0xBC, 0xFF}), want: ErrInvalidElementSize},
		{name: "child exceeds TrackEntry", data: wrap([]byte{0x4A, 0xBC, 0x81}), want: ErrInvalidElementSize},
		{name: "child exceeds Tracks", data: testElement(segmentElement, testElement(tracksElement, []byte{0xAE, 0x81})), want: ErrInvalidElementSize},
		{name: "child exceeds Segment", data: testElement(segmentElement, []byte{0xEC, 0x81}), want: ErrInvalidElementSize},
		{name: "string limit", data: wrap(testElement(nameElement, make([]byte, maxTrackStringSize+1))), want: ErrInvalidElementSize},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entries, err := parseTracks(bytes.NewReader(tc.data))
			if !errors.Is(err, tc.want) || entries != nil {
				t.Fatalf("parseTracks() = %+v, %v; want nil, %v", entries, err, tc.want)
			}
		})
	}
}

func TestParseTracksEveryTruncation(t *testing.T) {
	data := testElement(segmentElement, testElement(tracksElement, subtitleTrack(1)))
	for length := 1; length < len(data); length++ {
		entries, err := parseTracks(bytes.NewReader(data[:length]))
		if err == nil || entries != nil {
			t.Fatalf("length %d: parseTracks() = %+v, %v; want nil, error", length, entries, err)
		}
	}
}

func FuzzParseTracks(f *testing.F) {
	f.Add(testElement(segmentElement, testElement(tracksElement, subtitleTrack(1))))
	f.Add([]byte{0x18, 0x53, 0x80, 0x67, 0xFF, 0x1F, 0x43, 0xB6, 0x75, 0xFF})
	f.Fuzz(func(t *testing.T, data []byte) {
		entries, err := parseTracks(bytes.NewReader(data))
		if err != nil && entries != nil {
			t.Fatal("invalid file returned partial tracks")
		}
	})
}
