package nfo

import (
	"encoding/xml"
	"errors"
	"io"

	polochon "github.com/odwrtw/polochon/lib"
)

// Custom error
var (
	ErrInvalidType = errors.New("nfo: invalid type")
)

// Read reads the NFO from a reader
func Read(r io.Reader, i interface{}) error {
	var nfo interface{}

	switch t := i.(type) {
	case *polochon.Movie:
		nfo = NewMovie(t)
	case *polochon.Show:
		nfo = NewShow(t)
	case *polochon.ShowEpisode:
		nfo = NewEpisode(t)
	default:
		return ErrInvalidType
	}

	return xml.NewDecoder(r).Decode(nfo)
}

// Write writes the NFO into a writer
func Write(w io.Writer, i interface{}) error {
	var nfo interface{}

	switch t := i.(type) {
	case *polochon.Movie:
		nfo = NewMovie(t)
	case *polochon.Show:
		nfo = NewShow(t)
	case *polochon.ShowEpisode:
		nfo = NewEpisode(t)
	default:
		return ErrInvalidType
	}

	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	return enc.Encode(nfo)
}
