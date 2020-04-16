package papi

import (
	"fmt"

	polochon "github.com/odwrtw/polochon/lib"
)

// Season represents a season
type Season struct {
	ShowImdbID string           `json:"show_imdb_id"`
	Season     int              `json:"season"`
	Episodes   map[int]*Episode `json:"-"`
}

// uri implements the Resource interface
func (s *Season) uri() (string, error) {
	if s.ShowImdbID == "" {
		return "", ErrMissingShowImdbID
	}

	if s.Season == 0 {
		return "", ErrMissingSeason
	}

	return fmt.Sprintf("shows/%s/seasons/%d", s.ShowImdbID, s.Season), nil
}

// getDetails implements the resource interface
func (s *Season) getDetails(c *Client) error {
	return c.getSeasonDetails(s)
}

// getSeasonDetails updates the season with detailed informations from polochon
func (c *Client) getSeasonDetails(s *Season) error {
	uri, err := s.uri()
	if err != nil {
		return err
	}

	if s.Episodes == nil {
		s.Episodes = map[int]*Episode{}
	}

	type Input struct {
		*Season
		EpisodeList []int `json:"episodes"`
	}
	input := Input{Season: s}

	url := fmt.Sprintf("%s/%s", c.endpoint, uri)
	if err := c.get(url, &input); err != nil {
		return err
	}

	for _, num := range input.EpisodeList {
		s.Episodes[num] = &Episode{ShowEpisode: &polochon.ShowEpisode{
			Season:     s.Season,
			Episode:    num,
			ShowImdbID: s.ShowImdbID,
		}}
	}

	return nil
}

// GetSeason returns the detailed infos from polochon about a season
func (c *Client) GetSeason(id string, season int) (*Season, error) {
	s := &Season{
		ShowImdbID: id,
		Season:     season,
	}

	if err := c.getSeasonDetails(s); err != nil {
		return nil, err
	}

	return s, nil
}
