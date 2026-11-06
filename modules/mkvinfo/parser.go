package mkvinfo

// Parser references:
//   - Matroska specifications: https://www.rfc-editor.org/rfc/rfc9559.html
//   - EBML encoding: https://www.rfc-editor.org/rfc/rfc8794.html
//   - Matroska element IDs: https://www.matroska.org/technical/elements.html

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"

	polochon "github.com/odwrtw/polochon/lib"
)

// Parser errors
var (
	ErrNotAVideo          = errors.New("mkvinfo: not a video")
	ErrMissingFile        = errors.New("mkvinfo: missing file")
	ErrMissingFilePath    = errors.New("mkvinfo: missing file path")
	ErrInvalidElement     = errors.New("mkvinfo: invalid element")
	ErrInvalidElementSize = errors.New("mkvinfo: invalid element size")
)

const (
	idSegment      = 0x18538067
	idCluster      = 0x1F43B675
	idTracks       = 0x1654AE6B
	idTrackEntry   = 0xAE
	idTrackType    = 0x83
	idCodecID      = 0x86
	idLanguage     = 0x22B59C
	idLanguageIETF = 0x22B59D
	idName         = 0x536E
	idFlagForced   = 0x55AA

	// Only strings we actually use are buffered. Unrelated payloads have no
	// allocation limit because they are skipped, not decoded.
	maxTrackStringSize = 64 * 1024
)

// readVINT reads an EBML variable-size integer without its length marker.
// IDs are at most four bytes; element sizes are at most eight bytes.
func readVINT(r io.Reader, maxWidth int) (uint64, int, error) {
	var data [8]byte
	if _, err := io.ReadFull(r, data[:1]); err != nil {
		return 0, 0, err
	}
	width, marker := 1, byte(0x80)
	for marker != 0 && data[0]&marker == 0 {
		width++
		marker >>= 1
	}
	if width > maxWidth {
		return 0, 0, ErrInvalidElement
	}
	if _, err := io.ReadFull(r, data[1:width]); err != nil {
		return 0, 0, unexpectedEOF(err)
	}
	value := uint64(data[0] &^ marker)
	for _, b := range data[1:width] {
		value = value<<8 | uint64(b)
	}
	return value, width, nil
}

// readElement returns a payload size of -1 for an unknown-size element.
// Limited readers enforce each known-size parent's boundary, including EOF
// between children: physical EOF before that boundary is truncation.
func readElement(r io.Reader) (uint64, int64, error) {
	id, width, err := readVINT(r, 4)
	if err != nil {
		if parent, ok := r.(*io.LimitedReader); ok && parent.N > 0 {
			err = unexpectedEOF(err)
		}
		return 0, 0, err
	}
	if id == uint64(1)<<(7*width)-1 {
		return 0, 0, ErrInvalidElement
	}
	id |= uint64(1) << (7 * width)
	size, width, err := readVINT(r, 8)
	if err != nil {
		return 0, 0, unexpectedEOF(err)
	}
	if size == uint64(1)<<(7*width)-1 {
		return id, -1, nil
	}
	if parent, ok := r.(*io.LimitedReader); ok && int64(size) > parent.N {
		return 0, 0, ErrInvalidElementSize
	}
	return id, int64(size), nil
}

func unexpectedEOF(err error) error {
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
}

func skipElement(r io.Reader, size int64) error {
	if size < 0 {
		return ErrInvalidElementSize
	}
	_, err := io.CopyN(io.Discard, r, size)
	return unexpectedEOF(err)
}

func parseTracks(r io.Reader) ([]*TrackEntry, error) {
	for {
		id, size, err := readElement(r)
		if err == io.EOF {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		if id == idSegment {
			if size >= 0 {
				r = &io.LimitedReader{R: r, N: size}
			}
			return parseSegment(r)
		}
		if err := skipElement(r, size); err != nil {
			return nil, err
		}
	}
}

func parseSegment(r io.Reader) ([]*TrackEntry, error) {
	for {
		id, size, err := readElement(r)
		if err == io.EOF {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		// Segment and Cluster are the only Matroska containers that allow
		// unknown sizes. Walk an unknown Cluster's direct children until the
		// next sibling; their known-size payloads remain opaque.
		if id == idCluster && size < 0 {
			continue
		}
		if size < 0 {
			return nil, ErrInvalidElementSize
		}
		if id == idTracks {
			return parseTrackEntries(&io.LimitedReader{R: r, N: size})
		}
		if err := skipElement(r, size); err != nil {
			return nil, err
		}
	}
}

func parseTrackEntries(r *io.LimitedReader) ([]*TrackEntry, error) {
	var entries []*TrackEntry
	for {
		id, size, err := readElement(r)
		if err == io.EOF {
			return entries, nil
		}
		if err != nil {
			return nil, err
		}
		if size < 0 {
			return nil, ErrInvalidElementSize
		}
		if id != idTrackEntry {
			if err := skipElement(r, size); err != nil {
				return nil, err
			}
			continue
		}
		entry, err := parseTrackEntry(&io.LimitedReader{R: r, N: size})
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
}

func parseTrackEntry(r *io.LimitedReader) (*TrackEntry, error) {
	entry := &TrackEntry{Language: "eng"} // Matroska's default.
	for {
		id, size, err := readElement(r)
		if err == io.EOF {
			return entry, nil
		}
		if err != nil {
			return nil, err
		}
		if size < 0 {
			return nil, ErrInvalidElementSize
		}
		switch id {
		case idTrackType, idFlagForced:
			if size > 8 {
				return nil, ErrInvalidElementSize
			}
			var data [8]byte
			if _, err := io.ReadFull(r, data[8-size:]); err != nil {
				return nil, unexpectedEOF(err)
			}
			value := binary.BigEndian.Uint64(data[:])
			if id == idTrackType {
				entry.Type = TrackType(value)
			} else {
				entry.Forced = value == 1
			}
		case idName, idCodecID, idLanguage, idLanguageIETF:
			if size > maxTrackStringSize {
				return nil, ErrInvalidElementSize
			}
			data := make([]byte, int(size))
			if _, err := io.ReadFull(r, data); err != nil {
				return nil, unexpectedEOF(err)
			}
			// EBML strings may be terminated and padded with null bytes.
			if end := bytes.IndexByte(data, 0); end >= 0 {
				data = data[:end]
			}
			value := string(data)
			switch id {
			case idName:
				entry.Name = value
			case idCodecID:
				entry.Codec = value
			case idLanguage:
				entry.Language = value
			case idLanguageIETF:
				entry.LanguageIETF = &value
			}
		default:
			if err := skipElement(r, size); err != nil {
				return nil, err
			}
		}
	}
}

// ParseFile parses the mkv file to find entries
func ParseFile(file *polochon.File) ([]*TrackEntry, error) {
	if file == nil {
		return nil, ErrMissingFile
	}

	if file.Path == "" {
		return nil, ErrMissingFilePath
	}

	// This module only works with mkv files
	if file.Ext() != ".mkv" {
		return nil, polochon.ErrNotAvailable
	}

	f, err := os.Open(file.Path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	return parseTracks(f)
}
