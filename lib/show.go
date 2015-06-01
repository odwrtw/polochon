package polochon

import (
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
)

// Show errors
var (
	ErrMissingShowImageURL = errors.New("show: missing URL to download show images")
)

// Show represents a tv show
type Show struct {
	XMLName   xml.Name       `xml:"tvshow" json:"-"`
	Title     string         `xml:"title" json:"title"`
	ShowTitle string         `xml:"showtitle" json:"-"`
	Rating    float32        `xml:"rating" json:"rating"`
	Plot      string         `xml:"plot" json:"plot"`
	URL       string         `xml:"episodeguide>url" json:"-"`
	TvdbID    int            `xml:"tvdbid" json:"tvdb_id"`
	ImdbID    string         `xml:"imdbid" json:"imdb_id"`
	Year      int            `xml:"year" json:"year"`
	Banner    string         `xml:"-" json:"banner"`
	Fanart    string         `xml:"-" json:"fanart"`
	Poster    string         `xml:"-" json:"poster"`
	Episodes  []*ShowEpisode `xml:"-" json:"episodes"`
	config    *ShowConfig
	log       *logrus.Entry
}

// NewShow returns a new show
func NewShow() *Show {
	return &Show{
		XMLName: xml.Name{Space: "", Local: "tvshow"},
	}
}

// readShowNFO deserialized a XML file into a ShowSeason
func readShowNFO(r io.Reader) (*Show, error) {
	s := &Show{}

	if err := readNFO(r, s); err != nil {
		return nil, err
	}

	return s, nil
}

// GetDetails helps getting infos for a show
func (s *Show) GetDetails() error {
	var err error
	for _, d := range s.config.Detailers {
		err = d.GetDetails(s)
		if err == nil {
			break
		}
		s.log.Warnf("failed to get details from detailer: %q", err)
	}
	return err
}

// storePath returns the show store path from the config
func (s *Show) storePath() string {
	return filepath.Join(s.config.Dir, s.ShowTitle)
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
