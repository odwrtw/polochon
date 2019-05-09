package nfo

import (
	"encoding/xml"

	polochon "github.com/odwrtw/polochon/lib"
)

// Movie represents a movie NFO
type Movie struct {
	*polochon.Movie
}

// NewMovie returns a MovieNFO from a Movie
func NewMovie(m *polochon.Movie) *Movie {
	return &Movie{Movie: m}
}

// movieFields represents the fields in the NFO file
type movieFields struct {
	Metadata      Metadata `xml:"polochon"`
	ImdbID        string   `xml:"id"`
	OriginalTitle string   `xml:"originaltitle"`
	Plot          string   `xml:"plot"`
	Rating        float32  `xml:"rating"`
	Runtime       int      `xml:"runtime"`
	SortTitle     string   `xml:"sorttitle"`
	Tagline       string   `xml:"tagline"`
	Thumb         string   `xml:"thumb"`
	Fanart        string   `xml:"customfanart"`
	Title         string   `xml:"title"`
	TmdbID        int      `xml:"tmdbid"`
	Votes         int      `xml:"votes"`
	Year          int      `xml:"year"`
	Genres        []string `xml:"genre"`
}

// MarshalXML implements the XML Marshaler interface
func (m *Movie) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Space: "", Local: "movie"}

	nfo := &movieFields{
		Metadata:      Metadata{&m.VideoMetadata},
		ImdbID:        m.ImdbID,
		OriginalTitle: m.OriginalTitle,
		Plot:          m.Plot,
		Rating:        m.Rating,
		Runtime:       m.Runtime,
		SortTitle:     m.SortTitle,
		Tagline:       m.Tagline,
		Thumb:         m.Thumb,
		Fanart:        m.Fanart,
		Title:         m.Title,
		TmdbID:        m.TmdbID,
		Votes:         m.Votes,
		Year:          m.Year,
		Genres:        m.Genres,
	}

	return e.EncodeElement(nfo, start)
}

// UnmarshalXML implements the XML Unmarshaler interface
func (m *Movie) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	nfo := movieFields{Metadata: Metadata{&polochon.VideoMetadata{}}}
	if err := d.DecodeElement(&nfo, &start); err != nil {
		return err
	}

	m.VideoMetadata = *nfo.Metadata.VideoMetadata
	m.ImdbID = nfo.ImdbID
	m.OriginalTitle = nfo.OriginalTitle
	m.Plot = nfo.Plot
	m.Rating = nfo.Rating
	m.Runtime = nfo.Runtime
	m.SortTitle = nfo.SortTitle
	m.Tagline = nfo.Tagline
	m.Thumb = nfo.Thumb
	m.Fanart = nfo.Fanart
	m.Title = nfo.Title
	m.TmdbID = nfo.TmdbID
	m.Votes = nfo.Votes
	m.Year = nfo.Year
	m.Genres = nfo.Genres

	return nil
}
