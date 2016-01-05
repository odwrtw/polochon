package polochon

import "encoding/xml"

// MovieNFO represents a movie NFO in kodi
type MovieNFO struct {
	*Movie
}

// NewMovieNFO returns a MovieNFO from a Movie
func NewMovieNFO(m *Movie) *MovieNFO {
	return &MovieNFO{Movie: m}
}

// NFO represents a show NFO in kodi
type movieNFODetails struct {
	ImdbID        string  `xml:"id"`
	OriginalTitle string  `xml:"originaltitle"`
	Plot          string  `xml:"plot"`
	Rating        float32 `xml:"rating"`
	Runtime       int     `xml:"runtime"`
	SortTitle     string  `xml:"sorttitle"`
	Tagline       string  `xml:"tagline"`
	Thumb         string  `xml:"thumb"`
	Fanart        string  `xml:"customfanart"`
	Title         string  `xml:"title"`
	TmdbID        int     `xml:"tmdbid"`
	Votes         int     `xml:"votes"`
	Year          int     `xml:"year"`
}

// MarshalXML implements the XML Marshaler interface
func (m *MovieNFO) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Space: "", Local: "movie"}

	nfo := &movieNFODetails{
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
	}

	return e.EncodeElement(nfo, start)
}

// UnmarshalXML implements the XML Unmarshaler interface
func (m *MovieNFO) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	nfo := movieNFODetails{}
	if err := d.DecodeElement(&nfo, &start); err != nil {
		return err
	}

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

	return nil
}
