package polochon

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/Sirupsen/logrus"
)

// Show episode errors
var (
	ErrMissingShowEpisodeFilePath = errors.New("show episode: missing file path")
	ErrMissingShowEpisodeDir      = errors.New("show episode: missing show dir in config")
)

// ShowEpisode represents a tvshow episode
type ShowEpisode struct {
	XMLName       xml.Name  `xml:"episodedetails"`
	Title         string    `xml:"title"`
	ShowTitle     string    `xml:"showtitle"`
	Season        int       `xml:"season"`
	Episode       int       `xml:"episode"`
	TvdbID        int       `xml:"uniqueid"`
	Aired         string    `xml:"aired"`
	Plot          string    `xml:"plot"`
	Runtime       int       `xml:"runtime"`
	Thumb         string    `xml:"thumb"`
	Rating        float32   `xml:"rating"`
	ShowImdbID    string    `xml:"showimdbid"`
	ShowTvdbID    int       `xml:"showtvdbid"`
	EpisodeImdbID string    `xml:"episodeimdbid"`
	Torrents      []Torrent `xml:"-"`
	File          *File     `xml:"-"`
	Show          *Show     `xml:"-"`
	config        *ShowConfig
	log           *logrus.Entry
}

// NewShowEpisode returns a new show episode
func NewShowEpisode() *ShowEpisode {
	return &ShowEpisode{
		XMLName: xml.Name{Space: "", Local: "episodedetails"},
	}
}

// Type implements the video interface
func (s *ShowEpisode) Type() VideoType {
	return ShowEpisodeType
}

// SetFile implements the video interface
func (s *ShowEpisode) SetFile(f *File) {
	s.File = f
}

// SetConfig implements the video interface
func (s *ShowEpisode) SetConfig(c *Config) {
	// Only keep show config
	s.config = &c.Show

	// Set logger
	s.log = c.Log.WithFields(logrus.Fields{
		"type": "show_episode",
	})
}

// readShowEpisodeNFO deserialized a XML file into a ShowEpisode
func readShowEpisodeNFO(r io.Reader) (*ShowEpisode, error) {
	s := &ShowEpisode{}

	if err := readNFO(r, s); err != nil {
		return nil, err
	}

	return s, nil
}

// GetDetails helps getting infos for a show
func (s *ShowEpisode) GetDetails() error {
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

// GetTorrents helps getting the torrent files for a movie
func (s *ShowEpisode) GetTorrents() error {
	var err error
	for _, t := range s.config.Torrenters {
		err = t.GetTorrents(s)
		if err == nil {
			break
		}

		s.log.Warnf("failed to get torrents from torrenter: %q", err)
	}
	return err
}

//  create the show nfo if it doesn't exists yet
func (s *ShowEpisode) storePath() string {
	return filepath.Join(s.config.Dir, s.ShowTitle, fmt.Sprintf("Season %d", s.Season))
}

// createShowDir creates a new show and stores it
func (s *ShowEpisode) createShowDir() error {
	show := &Show{
		Title:     s.ShowTitle,
		ShowTitle: s.ShowTitle,
		TvdbID:    s.ShowTvdbID,
		ImdbID:    s.ShowImdbID,
		config:    s.config,
		log:       s.log,
	}

	// If the show nfo does not exist yet, create it
	if _, err := os.Stat(show.nfoPath()); os.IsNotExist(err) {
		// Get details
		if err := show.GetDetails(); err != nil {
			return err
		}

		// Store it
		if err := show.Store(); err != nil {
			return err
		}
	}

	return nil
}

//  create the show nfo if it doesn't exists yet
func (s *ShowEpisode) createShowSeasonDir() error {
	seasonDir := s.storePath()

	if _, err := os.Stat(seasonDir); os.IsNotExist(err) {
		s.log.Debugf("Show season folder does not exist, let's create one: %q", seasonDir)

		// Create folder
		if err = os.Mkdir(seasonDir, os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

//  move the episode to the season dir
func (s *ShowEpisode) move() error {
	storePath := s.storePath()

	// If the show episode already in the right dir there is nothing to do
	if path.Dir(s.File.Path) == storePath {
		s.log.Debug("show episode already in the destination folder")
		return nil
	}

	// Move the episode into the folder
	newPath := filepath.Join(storePath, path.Base(s.File.Path))
	s.log.Debug("Moving episode to folder")
	s.log.Debugf("Old path: %q", s.File.Path)
	s.log.Debugf("New path: %q", newPath)
	if err := os.Rename(s.File.Path, newPath); err != nil {
		return err
	}

	// Set the new movie path
	s.File.Path = newPath

	return nil
}

// Store create the show episode nfo and download the images
func (s *ShowEpisode) Store() error {
	if s.File == nil {
		return ErrMissingShowEpisodeFilePath
	}

	if s.config == nil || s.config.Dir == "" {
		return ErrMissingShowEpisodeDir
	}

	s.log = s.log.WithFields(logrus.Fields{
		"function":   "store",
		"show_title": s.ShowTitle,
		"season":     s.Season,
		"episode":    s.Episode,
	})

	// Create show dir if needed
	if err := s.createShowDir(); err != nil {
		return err
	}

	// Create show season dir if necessary
	if err := s.createShowSeasonDir(); err != nil {
		return err
	}

	// Move the file
	if err := s.move(); err != nil {
		return err
	}

	// Create show NFO if necessary
	if err := MarshalInFile(s, s.File.NfoPath()); err != nil {
		return err
	}

	return nil
}
