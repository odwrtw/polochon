package papi

import (
	"fmt"

	polochon "github.com/odwrtw/polochon/lib"
)

// Subtitle represents a subtitle
type Subtitle struct {
	*polochon.Subtitle
}

func (s *Subtitle) uri() (string, error) {
	if s.Subtitle == nil {
		return "", ErrMissingSubtitle
	}

	if s.Video == nil {
		return "", ErrMissingSubtitleVideo
	}

	var r Resource
	switch v := s.Video.(type) {
	case *Movie:
		r = v
	case *Episode:
		r = v
	case *polochon.Movie:
		r = &Movie{Movie: v}
	case *polochon.ShowEpisode:
		r = &Episode{ShowEpisode: v}
	default:
		return "", ErrNotImplemented
	}

	uri, err := r.uri()
	if err != nil {
		return "", err
	}

	uri += "/subtitles/" + string(s.Lang)
	return uri, nil
}

func (s *Subtitle) getDetails(c *Client) error {
	return ErrNotImplemented
}

func (s *Subtitle) downloadURL() (string, error) {
	uri, err := s.uri()
	if err != nil {
		return "", err
	}

	return uri + "/download", nil
}

// SubtitleEntry represents a subtitle entry returned by the available subtitles listing.
type SubtitleEntry struct {
	Index int `json:"index"`
	*polochon.SubtitleEntry
}

// ListAvailableSubtitles returns the list of available subtitles for a video.
func (c *Client) ListAvailableSubtitles(video polochon.Video, lang polochon.Language) ([]*SubtitleEntry, error) {
	s := &Subtitle{Subtitle: &polochon.Subtitle{Video: video, Lang: lang}}
	uri, err := s.uri()
	if err != nil {
		return nil, err
	}

	var entries []*SubtitleEntry
	err = c.get(fmt.Sprintf("%s/%s/available", c.endpoint, uri), &entries)
	return entries, err
}

// DownloadSubtitleByIndex downloads a specific subtitle by its index from the available listing.
func (c *Client) DownloadSubtitleByIndex(video polochon.Video, lang polochon.Language, index int) (*Subtitle, error) {
	s := &Subtitle{Subtitle: &polochon.Subtitle{Video: video, Lang: lang}}
	uri, err := s.uri()
	if err != nil {
		return nil, err
	}

	result := &Subtitle{Subtitle: &polochon.Subtitle{}}
	err = c.post(fmt.Sprintf("%s/%s/available", c.endpoint, uri), struct {
		Index int `json:"index"`
	}{Index: index}, &result)
	return result, err
}

// UpdateSubtitle updates the subtitles of a ressource
func (c *Client) UpdateSubtitle(video polochon.Video, lang polochon.Language) (*Subtitle, error) {
	s := &Subtitle{Subtitle: &polochon.Subtitle{
		Video: video,
		Lang:  lang,
	}}

	uri, err := s.uri()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/%s", c.endpoint, uri)
	return s, c.post(url, nil, &s)
}
