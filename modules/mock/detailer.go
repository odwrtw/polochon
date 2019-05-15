package mock

import (
	"fmt"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// GetDetails implements the Detailer interface
func (mock *Mock) GetDetails(i interface{}, log *logrus.Entry) (err error) {
	switch v := i.(type) {
	case *polochon.Show:
		mock.getShowDetails(v)
	case *polochon.ShowEpisode:
		mock.getShowEpisodeDetails(v)
	case *polochon.Movie:
		mock.getMovieDetails(v)
	default:
		return ErrInvalidArgument
	}
	return err
}

func (mock *Mock) getMovieDetails(m *polochon.Movie) {
	if m.ImdbID == "" {
		m.ImdbID = randomImdbID()
	}
	if m.Title == "" {
		m.Title = fmt.Sprintf("Movie %s", m.ImdbID)
	}
	if m.TmdbID == 0 {
		m.TmdbID = 12345
	}
	if m.OriginalTitle == "" {
		m.OriginalTitle = m.Title
	}
	if m.SortTitle == "" {
		m.SortTitle = m.Title
	}
	if m.Thumb == "" {
		m.Thumb = fmt.Sprintf("http://base-photo.com/thumb/%s.jpg", m.ImdbID)
	}
	if m.Fanart == "" {
		m.Fanart = fmt.Sprintf("http://base-photo.com/fanart/%s.jpg", m.ImdbID)
	}

	m.Plot = fmt.Sprintf("This is the plot of the movie %s", m.Title)
	m.Rating = 5.0
	m.Runtime = 200
	m.Tagline = "What the fuck is a tagline"
	m.Votes = 1000
	m.Year = 2000
	m.Genres = []string{"Fucked up"}
}

func (mock *Mock) getShowEpisodeDetails(s *polochon.ShowEpisode) {
	// If we have infos from the show, fill up the episode
	if s.Show != nil {
		if s.Show.ImdbID != "" {
			s.ShowImdbID = s.Show.ImdbID
		}
		if s.Show.Title != "" {
			s.ShowTitle = s.Show.Title
		}
		if s.Show.TvdbID != 0 {
			s.ShowTvdbID = s.Show.TvdbID
		}
	}

	if s.ShowImdbID == "" {
		s.ShowImdbID = randomImdbID()
	}
	if s.Season == 0 {
		s.Season = 1
	}
	if s.Episode == 0 {
		s.Episode = 1
	}
	if s.Title == "" {
		s.Title = fmt.Sprintf("Title %s S%02dE%02d", s.ShowImdbID, s.Season, s.Episode)
	}
	if s.ShowTitle == "" {
		s.ShowTitle = fmt.Sprintf("Show %s", s.ShowImdbID)
	}
	if s.TvdbID == 0 {
		s.TvdbID = 123456
	}
	if s.ShowTvdbID == 0 {
		s.ShowTvdbID = 123456
	}
	if s.EpisodeImdbID == "" {
		s.EpisodeImdbID = randomImdbID()
	}
	if s.Thumb == "" {
		s.Thumb = fmt.Sprintf("http://base-photo.com/thumb/%s.jpg", s.ShowImdbID)
	}

	s.Aired = "Already aired"
	s.Plot = fmt.Sprintf("This is the plot of the episode %s.S%02dE%02d", s.ShowImdbID, s.Season, s.Episode)
	s.Runtime = 200
	s.Rating = 5
}

// getShowDetails will get some show details
func (mock *Mock) getShowDetails(s *polochon.Show) {
	if s.ImdbID == "" {
		s.ImdbID = randomImdbID()
	}
	if s.TvdbID == 0 {
		s.TvdbID = 12345
	}
	if s.Title == "" {
		s.Title = fmt.Sprintf("Show %s", s.ImdbID)
	}
	if s.Banner == "" {
		s.Banner = fmt.Sprintf("http://base-photo.com/banner/%s.jpg", s.ImdbID)
	}
	if s.Fanart == "" {
		s.Fanart = fmt.Sprintf("http://base-photo.com/fanart/%s.jpg", s.ImdbID)
	}
	if s.Poster == "" {
		s.Poster = fmt.Sprintf("http://base-photo.com/poster/%s.jpg", s.ImdbID)
	}

	s.Plot = fmt.Sprintf("This is the plot of the show of %s", s.Title)
	s.Year = 2000
	s.URL = fmt.Sprintf("http://movie-url.com/%s", s.Title)
	s.Rating = 5

	if s.Episodes != nil {
		return
	}

	nbSeasons := 5
	nbEpisodes := 10
	for i := 0; i < nbSeasons; i++ {
		for j := 0; j < nbEpisodes; j++ {
			// Create the episode
			episode := &polochon.ShowEpisode{
				ShowImdbID: s.ImdbID,
				ShowTitle:  s.Title,
				Season:     i,
				Episode:    j,
				ShowTvdbID: s.TvdbID,
			}
			// No need to check err because
			mock.getShowEpisodeDetails(episode)

			// Add the episode to the list
			s.Episodes = append(s.Episodes, episode)
		}
	}
}
