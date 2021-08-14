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

// UpdateSubtitles updates the subtitles of a ressource
func (c *Client) UpdateSubtitles(target Resource) ([]*Subtitle, error) {
	url, err := target.uri()
	if err != nil {
		return nil, err
	}

	var subtitles []*Subtitle
	return subtitles, c.post(fmt.Sprintf("%s/%s/subtitles", c.endpoint, url), nil, &subtitles)
}
