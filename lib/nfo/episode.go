package nfo

import (
	"encoding/xml"

	polochon "github.com/odwrtw/polochon/lib"
)

// Episode represents a show episode NFO
type Episode struct {
	*polochon.ShowEpisode
}

// NewEpisode returns a Episode from a ShowEpisode
func NewEpisode(se *polochon.ShowEpisode) *Episode {
	return &Episode{ShowEpisode: se}
}

// episodeFields represents the show fileds in the NFO file
type episodeFields struct {
	Metadata Metadata `xml:"polochon"`

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
func (e *Episode) MarshalXML(enc *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Space: "", Local: "episodedetails"}

	nfo := &episodeFields{
		Metadata: Metadata{
			VideoMetadata: &e.VideoMetadata,
		},
		Title:         e.Title,
		ShowTitle:     e.ShowTitle,
		Season:        e.Season,
		Episode:       e.Episode,
		TvdbID:        e.TvdbID,
		Aired:         e.Aired,
		Premiered:     e.Aired,
		Plot:          e.Plot,
		Runtime:       e.Runtime,
		Thumb:         e.Thumb,
		Rating:        e.Rating,
		ShowImdbID:    e.ShowImdbID,
		ShowTvdbID:    e.ShowTvdbID,
		EpisodeImdbID: e.EpisodeImdbID,
	}

	return enc.EncodeElement(nfo, start)
}

// UnmarshalXML implements the XML Unmarshaler interface
func (e *Episode) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	metadata := polochon.VideoMetadata{}
	nfo := episodeFields{
		Metadata: Metadata{VideoMetadata: &metadata},
	}
	if err := d.DecodeElement(&nfo, &start); err != nil {
		return err
	}

	e.VideoMetadata = *nfo.Metadata.VideoMetadata
	e.Subtitles = subFromMetadata(e.ShowEpisode, nfo.Metadata.EmbeddedSubtitles)
	e.Title = nfo.Title
	e.ShowTitle = nfo.ShowTitle
	e.Season = nfo.Season
	e.Episode = nfo.Episode
	e.TvdbID = nfo.TvdbID
	e.Aired = nfo.Aired
	e.Plot = nfo.Plot
	e.Runtime = nfo.Runtime
	e.Thumb = nfo.Thumb
	e.Rating = nfo.Rating
	e.ShowImdbID = nfo.ShowImdbID
	e.ShowTvdbID = nfo.ShowTvdbID
	e.EpisodeImdbID = nfo.EpisodeImdbID

	return nil
}
