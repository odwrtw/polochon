package nfo

import (
	"encoding/xml"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
)

// function to be overwritten during tests
var now = func() time.Time {
	return time.Now()
}

type metadataFields struct {
	DateAdded         string              `xml:"date_added"`
	Quality           string              `xml:"quality"`
	ReleaseGroup      string              `xml:"release_group"`
	AudioCodec        string              `xml:"audio_codec"`
	VideoCodec        string              `xml:"video_codec"`
	Container         string              `xml:"container"`
	EmbeddedSubtitles []polochon.Language `xml:"embedded_subtitles"`
}

// Metadata represents polochon's metadata
type Metadata struct {
	*polochon.VideoMetadata
}

// MarshalXML implements the XML Marshaler interface
func (m *Metadata) MarshalXML(enc *xml.Encoder, start xml.StartElement) error {
	if m.VideoMetadata == nil {
		return ErrInvalidType
	}

	if m.DateAdded.IsZero() {
		m.DateAdded = now().UTC().Truncate(time.Second)
	}

	nfo := &metadataFields{
		DateAdded:         now().UTC().Format(time.RFC3339),
		Quality:           string(m.Quality),
		ReleaseGroup:      m.ReleaseGroup,
		AudioCodec:        m.AudioCodec,
		VideoCodec:        m.VideoCodec,
		Container:         m.Container,
		EmbeddedSubtitles: m.EmbeddedSubtitles,
	}

	return enc.EncodeElement(nfo, start)
}

// UnmarshalXML implements the XML Unmarshaler interface
func (m *Metadata) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if m.VideoMetadata == nil {
		m.VideoMetadata = &polochon.VideoMetadata{}
	}

	nfo := metadataFields{}
	err := d.DecodeElement(&nfo, &start)
	if err != nil {
		return err
	}

	var dateAdded time.Time
	if nfo.DateAdded != "" {
		dateAdded, err = time.Parse(time.RFC3339, nfo.DateAdded)
		if err != nil {
			return err
		}
	}

	m.DateAdded = dateAdded.UTC()
	m.Quality = polochon.Quality(nfo.Quality)
	m.ReleaseGroup = nfo.ReleaseGroup
	m.AudioCodec = nfo.AudioCodec
	m.VideoCodec = nfo.VideoCodec
	m.Container = nfo.Container
	m.EmbeddedSubtitles = nfo.EmbeddedSubtitles

	return nil
}
