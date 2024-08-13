package papi

import (
	"fmt"

	polochon "github.com/odwrtw/polochon/lib"
)

// Episode struct returned by papi
type Episode struct {
	*polochon.ShowEpisode

	NFO       *File       `json:"nfo_file"`
	Subtitles []*Subtitle `json:"subtitles"`
}

// uri implements the Resource interface
func (e *Episode) uri() (string, error) {
	if e.ShowImdbID == "" {
		return "", ErrMissingShowEpisodeInformations
	}

	if e.Season == 0 || e.Episode == 0 {
		return "", ErrMissingShowEpisodeInformations
	}

	return fmt.Sprintf(
		"shows/%s/seasons/%d/episodes/%d",
		e.ShowImdbID, e.Season, e.Episode,
	), nil
}

// downloadURL implements the Downloadable interface
func (e *Episode) downloadURL() (string, error) {
	uri, err := e.uri()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/download", uri), nil
}

// getDetails implements the resource interface
func (e *Episode) getDetails(c *Client) error {
	return c.getEpisodeDetails(e)
}

// getEpisodeDetails updates the episode with detailed informations from polochon
func (c *Client) getEpisodeDetails(e *Episode) error {
	uri, err := e.uri()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s", c.endpoint, uri)
	return c.get(url, e)
}

// GetEpisode returns a new show episode with detailed informations
func (c *Client) GetEpisode(id string, season, episode int) (*Episode, error) {
	e := &Episode{ShowEpisode: &polochon.ShowEpisode{
		ShowImdbID: id,
		Season:     season,
		Episode:    episode,
	}}
	return e, c.getEpisodeDetails(e)
}
