package papi

import (
	"fmt"
	"strconv"

	polochon "github.com/odwrtw/polochon/lib"
	index "github.com/odwrtw/polochon/lib/media_index"
)

// Show struct returned by papi
type Show struct {
	*polochon.Show

	Fanart *File `json:"fanart_file"`
	Banner *File `json:"banner_file"`
	Poster *File `json:"poster_file"`
	NFO    *File `json:"nfo_file"`

	Seasons map[int]*Season `json:"-"`
}

func (s *Show) linkFiles() {
	for _, file := range []*File{s.Fanart, s.Banner, s.NFO, s.Poster} {
		if file == nil {
			continue
		}

		file.resource = s
	}
}

// uri implements the Resource interface
func (s *Show) uri() (string, error) {
	if s.Show == nil {
		return "", ErrMissingShow
	}

	if s.ImdbID == "" {
		return "", ErrMissingShowImdbID
	}

	return fmt.Sprintf("shows/%s", s.ImdbID), nil
}

func extractSeasons(imdbID string, input map[string]map[string]*index.Episode) (map[int]*Season, error) {
	ret := map[int]*Season{}

	for season, episodes := range input {
		sn, err := strconv.Atoi(season)
		if err != nil {
			return nil, err
		}

		s := &Season{
			ShowImdbID: imdbID,
			Season:     sn,
			Episodes:   map[int]*Episode{},
		}

		for episode, e := range episodes {
			en, err := strconv.Atoi(episode)
			if err != nil {
				return nil, err
			}

			pe := &polochon.ShowEpisode{
				ShowImdbID: imdbID,
				Episode:    en,
				Season:     sn,
			}
			pe.SetFile(polochon.File{
				Path: e.Filename,
				Size: e.Size,
			})
			pe.SetMetadata(&e.VideoMetadata)

			subs := []*Subtitle{}
			for _, s := range e.Subtitles {
				subs = append(subs, &Subtitle{
					Subtitle: &polochon.Subtitle{
						File:     polochon.File{Size: s.Size},
						Lang:     s.Lang,
						Embedded: s.Embedded,
						Video:    pe,
					},
				})
			}

			if len(subs) == 0 {
				subs = nil
			}

			newEpisode := &Episode{
				ShowEpisode: pe,
				Subtitles:   subs,
			}

			s.Episodes[en] = newEpisode
			newEpisode.NFO = NewFile(e.NFO, newEpisode)
		}

		ret[sn] = s
	}

	return ret, nil
}

// GetShows returns all the shows in the polochon library
func (c *Client) GetShows() (*ShowCollection, error) {
	url := fmt.Sprintf("%s/%s", c.endpoint, "shows")

	ids := map[string]struct {
		*Show
		Seasons map[string]map[string]*index.Episode `json:"seasons"`
	}{}

	var err error
	if err = c.get(url, &ids); err != nil {
		return nil, err
	}

	showCollection := NewShowCollection()
	for imdbID, data := range ids {
		data.ImdbID = imdbID
		data.linkFiles()

		data.Show.Seasons, err = extractSeasons(imdbID, data.Seasons)
		if err != nil {
			return nil, err
		}

		showCollection.Add(data.Show)
	}

	return showCollection, nil
}

// getDetails implements the resource interface
func (s *Show) getDetails(c *Client) error {
	return c.getShowDetails(s)
}

// HasSeason checks if the show has the season
func (s *Show) HasSeason(season int) bool {
	if s.Seasons == nil {
		return false
	}

	if _, ok := s.Seasons[season]; !ok {
		return false
	}

	return true
}

// HasEpisode checks if the show has an episode
func (s *Show) HasEpisode(season, episode int) bool {
	if !s.HasSeason(season) {
		return false
	}

	if s.Seasons[season].Episodes == nil {
		return false
	}

	for _, e := range s.Seasons[season].Episodes {
		if e.Episode == episode {
			return true
		}
	}

	return false
}

// getShowDetails updates the show with detailed informations from polochon
func (c *Client) getShowDetails(s *Show) error {
	uri, err := s.uri()
	if err != nil {
		return err
	}

	input := &struct {
		*Show
		Seasons map[string]map[string]*index.Episode `json:"seasons"`
	}{Show: s}

	url := fmt.Sprintf("%s/%s", c.endpoint, uri)
	if err := c.get(url, input); err != nil {
		return err
	}

	seasons, err := extractSeasons(s.ImdbID, input.Seasons)
	if err != nil {
		return err
	}
	s.Seasons = seasons

	return nil
}

// GetShow returns the detailed infos from polochon about a show
func (c *Client) GetShow(id string) (*Show, error) {
	s := &Show{Show: &polochon.Show{ImdbID: id}}
	if err := c.getShowDetails(s); err != nil {
		return nil, err
	}
	return s, nil
}

// GetEpisode checks if the show has an episode
func (s *Show) GetEpisode(season, episode int) *Episode {
	if !s.HasSeason(season) {
		return nil
	}

	if s.Seasons[season].Episodes == nil {
		return nil
	}

	for _, e := range s.Seasons[season].Episodes {
		if e.Episode == episode {
			return e
		}
	}

	return nil
}
