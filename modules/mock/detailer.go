package mock

import (
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
)

// Module constants
const (
	moduleName = "mock"
)

var (
	// ErrInvalidArgument is returned if the type is invalid
	ErrInvalidArgument = errors.New("mock: invalid argument type")
)

func init() {
	polochon.RegisterDetailer(moduleName, NewDetailer)
}

// Mock is a mock Detailer for test purposes
type Mock struct{}

// NewDetailer is an helper to avoid passing bytes
func NewDetailer(p []byte) (polochon.Detailer, error) {
	return &Mock{}, nil
}

// Name implements the Module interface
func (mock *Mock) Name() string {
	return moduleName
}

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
		m.ImdbID = "tt12345"
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

	m.Plot = fmt.Sprintf("This is the plot of the movie %s", m.Title)
	m.Rating = 5.0
	m.Runtime = 200
	m.Tagline = "What the fuck is a tagline"
	m.Votes = 1000
	m.Year = 2000
	m.Thumb = fmt.Sprintf("http://base-photo.com/thumb/%s.jpg", m.ImdbID)
	m.Fanart = fmt.Sprintf("http://base-photo.com/fanart/%s.jpg", m.ImdbID)
}

func (mock *Mock) getShowEpisodeDetails(s *polochon.ShowEpisode) {
	if s.ShowImdbID == "" {
		s.ShowImdbID = "tt12345"
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
		s.EpisodeImdbID = "tt123456"
	}

	s.Aired = "Already aired"
	s.Plot = fmt.Sprintf("This is the plot of the episode %s.S%02dE%02d", s.ShowImdbID, s.Season, s.Episode)
	s.Runtime = 200
	s.Thumb = fmt.Sprintf("http://base-photo.com/thumb/%s.jpg", s.ShowImdbID)
	s.Rating = 5
}

// getShowDetails will get some show details
func (mock *Mock) getShowDetails(s *polochon.Show) {
	if s.ImdbID == "" {
		s.ImdbID = "tt12345"
	}
	if s.TvdbID == 0 {
		s.TvdbID = 12345
	}
	if s.Title == "" {
		s.Title = fmt.Sprintf("Show %s", s.ImdbID)
	}
	s.Plot = fmt.Sprintf("This is the plot of the show of %s", s.Title)
	s.Year = 2000
	s.URL = fmt.Sprintf("http://movie-url.com/%s", s.Title)
	s.Rating = 5

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