package papi

import (
	"fmt"
	"strconv"

	polochon "github.com/odwrtw/polochon/lib"
)

// Show struct returned by papi
type Show struct {
	*polochon.Show

	Seasons map[int]*Season `json:"-"`
}

// SidecarFiles returns the show sidecar files as downloadable papi.File objects.
func (s *Show) SidecarFiles() []*File {
	return []*File{
		NewFile(s.FanartFile, s),
		NewFile(s.BannerFile, s),
		NewFile(s.PosterFile, s),
		NewFile(s.NFOFile, s),
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

func extractSeasons(imdbID string, input map[string]map[string]*polochon.ShowEpisode) (map[int]*Season, error) {
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

			if e == nil {
				continue
			}

			e.ShowImdbID = imdbID
			e.Episode = en
			e.Season = sn
			// Name holds the video filename from the HTTP response; use it as the path.
			if e.Name != "" {
				e.Path = e.Name
			}

			subs := []*Subtitle{}
			for _, sub := range e.Subtitles {
				sub.Video = e
				subs = append(subs, &Subtitle{Subtitle: sub})
			}
			if len(subs) == 0 {
				subs = nil
			}

			newEpisode := &Episode{
				ShowEpisode: e,
				Subtitles:   subs,
			}
			newEpisode.NFO = NewFile(e.NFOFile, newEpisode)

			s.Episodes[en] = newEpisode
		}

		ret[sn] = s
	}

	return ret, nil
}

// GetShows returns all the shows in the polochon library
func (c *Client) GetShows() (*ShowCollection, error) {
	url := fmt.Sprintf("%s/%s", c.endpoint, "shows")

	ids := map[string]*struct {
		*Show
		Seasons map[string]map[string]*polochon.ShowEpisode `json:"seasons"`
	}{}

	var err error
	if err = c.get(url, &ids); err != nil {
		return nil, err
	}

	showCollection := NewShowCollection()
	for imdbID, data := range ids {
		if data.Show == nil {
			data.Show = &Show{Show: &polochon.Show{}}
		}
		if data.Show.Show == nil {
			data.Show.Show = &polochon.Show{}
		}
		data.ImdbID = imdbID

		data.Show.Seasons, err = extractSeasons(imdbID, data.Seasons)
		if err != nil {
			return nil, err
		}

		if err = showCollection.Add(data.Show); err != nil {
			return nil, err
		}
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
		Seasons map[string]map[string]*polochon.ShowEpisode `json:"seasons"`
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
