package nfo

import (
	"encoding/xml"
	"time"

	"github.com/odwrtw/polochon/lib"
)

// Show represents a show NFO
type Show struct {
	*polochon.Show
}

// NewShow returns a Show from a Show
func NewShow(s *polochon.Show) *Show {
	return &Show{Show: s}
}

// showFields represents the show fileds in the NFO file
type showFields struct {
	Title     string  `xml:"title"`
	ShowTitle string  `xml:"showtitle"`
	Rating    float32 `xml:"rating"`
	Plot      string  `xml:"plot"`
	URL       string  `xml:"episodeguide>url"`
	TvdbID    int     `xml:"tvdbid"`
	ImdbID    string  `xml:"imdbid"`
	Year      int     `xml:"year"`
	Premiered string  `xml:"premiered"`
}

// MarshalXML implements the XML Marshaler interface
func (s *Show) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Space: "", Local: "tvshow"}

	nfo := &showFields{
		Title:     s.Title,
		ShowTitle: s.Title,
		Rating:    s.Rating,
		Plot:      s.Plot,
		URL:       s.URL,
		TvdbID:    s.TvdbID,
		ImdbID:    s.ImdbID,
		Year:      s.Year,
	}

	if s.FirstAired != nil {
		nfo.Premiered = s.FirstAired.Format("2006-01-02")
	}

	return e.EncodeElement(nfo, start)
}

// UnmarshalXML implements the XML Unmarshaler interface
func (s *Show) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	nfo := showFields{}
	if err := d.DecodeElement(&nfo, &start); err != nil {
		return err
	}

	s.Title = nfo.Title
	s.Rating = nfo.Rating
	s.Plot = nfo.Plot
	s.URL = nfo.URL
	s.TvdbID = nfo.TvdbID
	s.ImdbID = nfo.ImdbID
	s.Year = nfo.Year

	if nfo.Premiered != "" {
		firstAired, err := time.Parse("2006-01-02", nfo.Premiered)
		if err != nil {
			return err
		}

		s.FirstAired = &firstAired
	}

	return nil
}
