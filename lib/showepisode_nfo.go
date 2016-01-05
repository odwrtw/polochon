package polochon

import "encoding/xml"

// ShowEpisodeNFO represents a tvshow episode
type ShowEpisodeNFO struct {
	*ShowEpisode
}

// NewShowEpisodeNFO returns a ShowEpisodeNFO from a Show
func NewShowEpisodeNFO(se *ShowEpisode) *ShowEpisodeNFO {
	return &ShowEpisodeNFO{ShowEpisode: se}
}

type showEpisodeNFODetails struct {
	Title         string  `xml:"title"`
	ShowTitle     string  `xml:"showtitle"`
	Season        int     `xml:"season"`
	Episode       int     `xml:"episode"`
	TvdbID        int     `xml:"uniqueid"`
	Aired         string  `xml:"aired"`
	Premiered     string  `xml:"premiered"`
	Plot          string  `xml:"plot"`
	Runtime       int     `xml:"runtime"`
	Thumb         string  `xml:"thumb"`
	Rating        float32 `xml:"rating"`
	ShowImdbID    string  `xml:"showimdbid"`
	ShowTvdbID    int     `xml:"showtvdbid"`
	EpisodeImdbID string  `xml:"episodeimdbid"`
}

// MarshalXML implements the XML Marshaler interface
func (se *ShowEpisodeNFO) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Space: "", Local: "episodedetails"}

	nfo := &showEpisodeNFODetails{
		Title:         se.Title,
		ShowTitle:     se.ShowTitle,
		Season:        se.Season,
		Episode:       se.Episode,
		TvdbID:        se.TvdbID,
		Aired:         se.Aired,
		Premiered:     se.Aired,
		Plot:          se.Plot,
		Runtime:       se.Runtime,
		Thumb:         se.Thumb,
		Rating:        se.Rating,
		ShowImdbID:    se.ShowImdbID,
		ShowTvdbID:    se.ShowTvdbID,
		EpisodeImdbID: se.EpisodeImdbID,
	}

	return e.EncodeElement(nfo, start)
}

// UnmarshalXML implements the XML Unmarshaler interface
func (se *ShowEpisodeNFO) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	nfo := showEpisodeNFODetails{}
	if err := d.DecodeElement(&nfo, &start); err != nil {
		return err
	}

	se.Title = nfo.Title
	se.ShowTitle = nfo.ShowTitle
	se.Season = nfo.Season
	se.Episode = nfo.Episode
	se.TvdbID = nfo.TvdbID
	se.Aired = nfo.Aired
	se.Plot = nfo.Plot
	se.Runtime = nfo.Runtime
	se.Thumb = nfo.Thumb
	se.Rating = nfo.Rating
	se.ShowImdbID = nfo.ShowImdbID
	se.ShowTvdbID = nfo.ShowTvdbID
	se.EpisodeImdbID = nfo.EpisodeImdbID

	return nil
}
