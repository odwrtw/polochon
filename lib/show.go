package polochon

import (
	"encoding/xml"
	"io"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/errors"
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
	Banner     string         `json:"banner"`
	Fanart     string         `json:"fanart"`
	Poster     string         `json:"poster"`
	Episodes   []*ShowEpisode `json:"episodes"`
}

// MarshalXML implements the XML Marshaler interface
func (s *Show) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Space: "", Local: "tvshow"}

	nfo := &ShowNFO{
		Title:     s.Title,
		ShowTitle: s.Title,
		Rating:    s.Rating,
		Plot:      s.Plot,
		URL:       s.URL,
		TvdbID:    s.TvdbID,
		ImdbID:    s.ImdbID,
		Year:      s.Year,
	}

	if s.FirstAired != nil {
		nfo.Premiered = s.FirstAired.Format("2006-01-02")
	}

	return e.EncodeElement(nfo, start)
}

// UnmarshalXML implements the XML Unmarshaler interface
func (s *Show) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	nfo := ShowNFO{}
	if err := d.DecodeElement(&nfo, &start); err != nil {
		return err
	}

	s.Title = nfo.Title
	s.Rating = nfo.Rating
	s.Plot = nfo.Plot
	s.URL = nfo.URL
	s.TvdbID = nfo.TvdbID
	s.ImdbID = nfo.ImdbID
	s.Year = nfo.Year

	if nfo.Premiered != "" {
		firstAired, err := time.Parse("2006-01-02", nfo.Premiered)
		if err != nil {
			return err
		}

		s.FirstAired = &firstAired
	}

	return nil
}

// NewShow returns a new show
func NewShow(showConf ShowConfig) *Show {
	return &Show{
		ShowConfig: showConf,
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
