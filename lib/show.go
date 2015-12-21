package polochon

import (
	"encoding/xml"
	"io"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/errors"
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

// GetDetails helps getting infos for a show
// If there is an error, it will be of type *errors.Collector
func (s *Show) GetDetails(log *logrus.Entry) error {
	c := errors.NewCollector()

	if len(s.Detailers) == 0 {
		c.Push(errors.Wrap("No detailer available").Fatal())
		return c
	}

	var done bool
	for _, d := range s.Detailers {
		err := d.GetDetails(s, log)
		if err == nil {
			done = true
			break
		}
		c.Push(errors.Wrap(err).Ctx("Detailer", d.Name()))
	}
	if !done {
		c.Push(errors.Wrap("All detailers failed").Fatal())
	}

	if c.HasErrors() {
		return c
	}

	return nil
}

// GetCalendar gets the calendar for the show
// If there is an error, it will be of type *errors.Error
func (s *Show) GetCalendar(log *logrus.Entry) (*ShowCalendar, *errors.Error) {
	if s.Calendar == nil {
		return nil, errors.Wrap("no show calendar fetcher configured").Fatal()
	}

	calendar, err := s.Calendar.GetShowCalendar(s, log)
	if err != nil {
		return nil, errors.Wrap(err).Fatal()
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
			Subtitlers: e.Subtitlers,
			Torrenters: e.Torrenters,
		},
	}
}
