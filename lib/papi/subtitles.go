package papi

import (
	"fmt"

	polochon "github.com/odwrtw/polochon/lib"
	index "github.com/odwrtw/polochon/lib/media_index"
)

// SubtitleURL returns the subtitle URL of a subtitlable content
func (c *Client) SubtitleURL(target Downloadable, lang polochon.Language) (string, error) {
	url, err := target.subtitleURL(lang)
	if err != nil {
		return "", err
	}

	return c.endpoint + "/" + url, nil
}

// UpdateSubtitles updates the subtitles of a ressource
func (c *Client) UpdateSubtitles(target Resource) ([]*index.Subtitle, error) {
	url, err := target.uri()
	if err != nil {
		return nil, err
	}

	var subtitles []*index.Subtitle
	return subtitles, c.post(fmt.Sprintf("%s/%s/subtitles", c.endpoint, url), nil, &subtitles)
}
