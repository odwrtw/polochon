package polochon

import (
	"time"

	"github.com/sirupsen/logrus"
)

// Show represents a tv show
type Show struct {
	ShowConfig `json:"-"`
	Title      string         `json:"title"`
	Rating     float32        `json:"rating"`
	Plot       string         `json:"plot"`
	URL        string         `json:"-"`
	TvdbID     int            `json:"tvdb_id"`
	ImdbID     string         `json:"imdb_id"`
	Year       int            `json:"year"`
	FirstAired *time.Time     `json:"first_aired"`
	Banner     string         `json:"-"`
	Fanart     string         `json:"-"`
	Poster     string         `json:"-"`
	Episodes   []*ShowEpisode `json:"-"`
}

// NewShow returns a new show
func NewShow(showConf ShowConfig) *Show {
	return &Show{
		ShowConfig: showConf,
	}
}

// GetCalendar gets the calendar for the show
// If there is an error, it will be of type *errors.Error
func (s *Show) GetCalendar(log *logrus.Entry) (*ShowCalendar, error) {
	if s.Calendar == nil {
		return nil, ErrCalendarModuleNotFound
	}

	return s.Calendar.GetShowCalendar(s, log)
}

// NewShowFromEpisode will return a show from an episode
func NewShowFromEpisode(e *ShowEpisode) *Show {
	return &Show{
		Title:  e.ShowTitle,
		TvdbID: e.ShowTvdbID,
		ImdbID: e.ShowImdbID,
		ShowConfig: ShowConfig{
			Detailers:  e.Detailers,
			Subtitlers: e.Subtitlers,
			Torrenters: e.Torrenters,
		},
	}
}
