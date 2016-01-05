package polochon

import (
	"encoding/xml"
	"errors"
	"io"
	"io/ioutil"
)

// Custom error
var (
	ErrInvalidArgument = errors.New("invalid argument")
)

// ReadNFO reads the NFO from a reader
func ReadNFO(r io.Reader, i interface{}) error {
	var nfo interface{}

	switch t := i.(type) {
	case *Movie:
		nfo = NewMovieNFO(t)
	case *Show:
		nfo = NewShowNFO(t)
	case *ShowEpisode:
		nfo = NewShowEpisodeNFO(t)
	default:
		return ErrInvalidArgument
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	return xml.Unmarshal(b, nfo)
}

// WriteNFO writes the NFO into a writer
func WriteNFO(w io.Writer, i interface{}) error {
	var nfo interface{}

	switch t := i.(type) {
	case *Movie:
		nfo = NewMovieNFO(t)
	case *Show:
		nfo = NewShowNFO(t)
	case *ShowEpisode:
		nfo = NewShowEpisodeNFO(t)
	default:
		return ErrInvalidArgument
	}

	b, err := xml.MarshalIndent(nfo, "", "  ")
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	if err != nil {
		return err
	}

	return nil
}
