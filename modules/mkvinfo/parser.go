package mkvinfo

import (
	"errors"
	"os"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/remko/go-mkvparse"
)

// Parser errors
var (
	ErrNotAVideo       = errors.New("mkvinfo: not a video")
	ErrMissingFile     = errors.New("mkvinfo: missing file")
	ErrMissingFilePath = errors.New("mkvinfo: missing file path")
)

type parser struct {
	current *TrackEntry
	entries []*TrackEntry
}

func (p *parser) HandleMasterBegin(id mkvparse.ElementID, info mkvparse.ElementInfo) (bool, error) {
	if id != mkvparse.TrackEntryElement {
		return true, nil
	}

	if p.current == nil {
		p.current = &TrackEntry{}
	}

	return true, nil
}

func (p *parser) HandleMasterEnd(id mkvparse.ElementID, info mkvparse.ElementInfo) error {
	if id != mkvparse.TrackEntryElement {
		return nil
	}

	if p.current != nil {
		if p.entries == nil {
			p.entries = []*TrackEntry{}
		}

		p.entries = append(p.entries, p.current)
		p.current = nil
	}
	return nil
}

func (p *parser) HandleString(id mkvparse.ElementID, value string, info mkvparse.ElementInfo) error {
	if p.current == nil {
		return nil
	}

	switch id {
	case mkvparse.LanguageElement:
		p.current.Language = value
	case mkvparse.CodecIDElement:
		p.current.Codec = value
	case mkvparse.NameElement:
		p.current.Name = value
	}

	return nil
}

func (p *parser) HandleInteger(id mkvparse.ElementID, value int64, info mkvparse.ElementInfo) error {
	if p.current == nil {
		return nil
	}

	if id == mkvparse.TrackTypeElement {
		p.current.Type = TrackType(value)
	}

	return nil
}

func (p *parser) HandleFloat(id mkvparse.ElementID, value float64, info mkvparse.ElementInfo) error {
	return nil
}

func (p *parser) HandleDate(id mkvparse.ElementID, value time.Time, info mkvparse.ElementInfo) error {
	return nil
}

func (p *parser) HandleBinary(id mkvparse.ElementID, value []byte, info mkvparse.ElementInfo) error {
	return nil
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
	defer f.Close()

	parser := parser{}
	err = mkvparse.ParseSections(f, &parser, mkvparse.TracksElement)
	if err != nil {
		return nil, err
	}

	return parser.entries, nil
}
