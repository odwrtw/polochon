package polochon

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/Sirupsen/logrus"
)

// Show represents a tv show
type Show struct {
	ShowConfig `xml:"-" json:"-"`
	XMLName    xml.Name       `xml:"tvshow" json:"-"`
	Title      string         `xml:"title" json:"title"`
	ShowTitle  string         `xml:"showtitle" json:"-"`
	Rating     float32        `xml:"rating" json:"rating"`
	Plot       string         `xml:"plot" json:"plot"`
	URL        string         `xml:"episodeguide>url" json:"-"`
	TvdbID     int            `xml:"tvdbid" json:"tvdb_id"`
	ImdbID     string         `xml:"imdbid" json:"imdb_id"`
	Year       int            `xml:"year" json:"year"`
	Banner     string         `xml:"-" json:"banner"`
	Fanart     string         `xml:"-" json:"fanart"`
	Poster     string         `xml:"-" json:"poster"`
	Episodes   []*ShowEpisode `xml:"-" json:"episodes"`
	log        *logrus.Entry
}

// NewShow returns a new show
func NewShow(showConf ShowConfig) *Show {
	return &Show{
		ShowConfig: showConf,
		XMLName:    xml.Name{Space: "", Local: "tvshow"},
	}
}

// readShowNFO deserialized a XML file into a ShowSeason
func readShowNFO(r io.Reader, conf ShowConfig) (*Show, error) {
	s := &Show{ShowConfig: conf}

	if err := readNFO(r, s); err != nil {
		return nil, err
	}

	return s, nil
}

// SetLogger sets the logger
func (s *Show) SetLogger(log *logrus.Entry) {
	s.log = log.WithField("type", "show")
}

// GetDetails helps getting infos for a show
func (s *Show) GetDetails() error {
	var err error
	for _, d := range s.Detailers {
		err = d.GetDetails(s, s.log)
		if err == nil {
			break
		}
		s.log.Warnf("failed to get details from detailer: %q", err)
	}
	return err
}

// GetCalendar gets the calendar for the show
func (s *Show) GetCalendar() (*ShowCalendar, error) {
	if s.Calendar == nil {
		return nil, fmt.Errorf("no show calendar fetcher configured")
	}

	calendar, err := s.Calendar.GetShowCalendar(s, s.log)
	if err != nil {
		return nil, err
	}

	return calendar, nil
}

// NewShowFromEpisode will return a show from an episode
func NewShowFromEpisode(e *ShowEpisode) *Show {
	return &Show{
		Title:     e.ShowTitle,
		ShowTitle: e.ShowTitle,
		TvdbID:    e.ShowTvdbID,
		ImdbID:    e.ShowImdbID,
		ShowConfig: ShowConfig{
			Detailers:  e.Detailers,
			Notifiers:  e.Notifiers,
			Subtitlers: e.Subtitlers,
			Torrenters: e.Torrenters,
		},
		log: e.log,
	}
}
