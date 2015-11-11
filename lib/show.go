package polochon

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/errors"
)

// Show errors
var (
	ErrMissingShowImageURL = errors.New("show: missing URL to download show images")
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
func (s *Show) GetDetails() (bool, *errors.Multiple) {
	merr := errors.NewMultiple()
	for _, d := range s.Detailers {
		err := d.GetDetails(s, s.log)
		if err == nil {
			return true, merr
		}
		merr.AddWithContext(err, errors.Context{
			"detailer": d.Name(),
		})
	}
	return false, merr
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

// storePath returns the show store path from the config
func (s *Show) storePath() string {
	return filepath.Join(s.Dir, s.ShowTitle)
}

// nfoPath returns the show nfo path
func (s *Show) nfoPath() string {
	return filepath.Join(s.storePath(), "tvshow.nfo")
}

// createShowDir create the show dir if it doesn't exists yet
func (s *Show) createShowDir() error {
	showDir := s.storePath()

	if _, err := os.Stat(showDir); os.IsNotExist(err) {
		s.log.Debugf("Show folder does not exist, let's create one: %q", showDir)

		// Create folder
		if err = os.Mkdir(showDir, os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

// createNFO create the show nfo if it doesn't exists yet
func (s *Show) createNFO() error {
	nfoPath := s.nfoPath()

	// Check if it exists
	if _, err := os.Stat(nfoPath); err == nil {
		return nil
	}

	// Write NFO into the file
	if err := MarshalInFile(s, nfoPath); err != nil {
		return err
	}

	s.log.Debug("show nfo saved to file")

	return nil
}

// Store create the show nfo and download the images
func (s *Show) Store() error {
	s.log = s.log.WithFields(logrus.Fields{
		"function": "store",
		"title":    s.ShowTitle,
	})

	// Create show dir if necessary
	if err := s.createShowDir(); err != nil {
		return err
	}

	// Create show NFO if necessary
	if err := s.createNFO(); err != nil {
		return err
	}

	// Download show images
	if err := s.downloadImages(); err != nil {
		return err
	}

	return nil
}

// Function to be overwritten during the tests
var downloadShowImage = func(URL, savePath string, log *logrus.Entry) error {
	return download(URL, savePath, log)
}

func (s *Show) downloadImages() error {
	if s.Fanart == "" || s.Banner == "" || s.Poster == "" {
		return ErrMissingShowImageURL
	}

	// Download images
	storePath := s.storePath()
	images := map[string]string{
		s.Fanart: filepath.Join(storePath, "banner.jpg"),
		s.Banner: filepath.Join(storePath, "fanart.jpg"),
		s.Poster: filepath.Join(storePath, "poster.jpg"),
	}
	for URL, savePath := range images {
		if err := downloadShowImage(URL, savePath, s.log); err != nil {
			return err
		}
	}

	return nil
}

// Delete implements the Video interface
func (s *Show) Delete() error {
	s.log.Infof("Removing Show %s", s.storePath())

	return os.RemoveAll(s.storePath())
}

// NewShowFromEpisode will return a show from an episode
func NewShowFromEpisode(e *ShowEpisode) *Show {
	return &Show{
		Title:     e.ShowTitle,
		ShowTitle: e.ShowTitle,
		TvdbID:    e.ShowTvdbID,
		ImdbID:    e.ShowImdbID,
		ShowConfig: ShowConfig{
			Dir:        e.Dir,
			Detailers:  e.Detailers,
			Notifiers:  e.Notifiers,
			Subtitlers: e.Subtitlers,
			Torrenters: e.Torrenters,
		},
		log: e.log,
	}
}
