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
	ShowConfig `xml:"-" json:"-"`
	File
	XMLName       xml.Name  `xml:"episodedetails" json:"-"`
	Title         string    `xml:"title" json:"title"`
	ShowTitle     string    `xml:"showtitle" json:"-"`
	Season        int       `xml:"season" json:"season"`
	Episode       int       `xml:"episode" json:"episode"`
	TvdbID        int       `xml:"uniqueid" json:"tvdb_id"`
	Aired         string    `xml:"aired" json:"aired"`
	Plot          string    `xml:"plot" json:"plot"`
	Runtime       int       `xml:"runtime" json:"runtime"`
	Thumb         string    `xml:"thumb" json:"thumb"`
	Rating        float32   `xml:"rating" json:"rating"`
	ShowImdbID    string    `xml:"showimdbid" json:"-"`
	ShowTvdbID    int       `xml:"showtvdbid" json:"-"`
	EpisodeImdbID string    `xml:"episodeimdbid" json:"imdb_id"`
	Torrents      []Torrent `xml:"-" json:"-"`
	Show          *Show     `xml:"-" json:"-"`
	log           *logrus.Entry
}

// PrepareForJSON return a copy of the object clean for the API
func (s ShowEpisode) PrepareForJSON() (ShowEpisode, error) {
	ok, err := filepath.Match(s.Dir+"/*/*/*", s.Path)
	if err != nil {
		return s, err
	}
	if !ok {
		return s, errors.New("Unexpected file path")
	}
	path, err := filepath.Rel(s.Dir, s.Path)
	if err != nil {
		return s, err
	}
	s.Path = path

	return s, nil
}

// NewShowEpisode returns a new show episode
func NewShowEpisode(showConf ShowConfig) *ShowEpisode {
	return &ShowEpisode{
		ShowConfig: showConf,
		XMLName:    xml.Name{Space: "", Local: "episodedetails"},
	}
}

// NewShowEpisodeFromFile returns a new show episode from a file
func NewShowEpisodeFromFile(showConf ShowConfig, file File) *ShowEpisode {
	return &ShowEpisode{
		ShowConfig: showConf,
		File:       file,
		XMLName:    xml.Name{Space: "", Local: "episodedetails"},
	}
}

// Type implements the video interface
func (s *ShowEpisode) Type() VideoType {
	return ShowEpisodeType
}

// SetFile implements the video interface
func (s *ShowEpisode) SetFile(f *File) {
	s.Path = f.Path
	s.ExcludeFileContaining = f.ExcludeFileContaining
	s.VideoExtentions = f.VideoExtentions
	s.AllowedExtentionsToDelete = f.AllowedExtentionsToDelete
	s.Guesser = f.Guesser
}

// SetConfig implements the video interface
func (s *ShowEpisode) SetConfig(c *VideoConfig, log *logrus.Logger) {
	s.Dir = c.Show.Dir
	s.Detailers = c.Show.Detailers
	s.Notifiers = c.Show.Notifiers
	s.Subtitlers = c.Show.Subtitlers
	s.Torrenters = c.Show.Torrenters

	// Set logger
	s.log = log.WithFields(logrus.Fields{
		"type": "show_episode",
	})
}

// readShowEpisodeNFO deserialized a XML file into a ShowEpisode
func readShowEpisodeNFO(r io.Reader, conf ShowConfig) (*ShowEpisode, error) {
	s := NewShowEpisode(conf)

	if err := readNFO(r, s); err != nil {
		return nil, err
	}

	return s, nil
}

// GetDetails helps getting infos for a show
func (s *ShowEpisode) GetDetails() error {
	var err error
	for _, d := range s.Detailers {
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
	for _, t := range s.Torrenters {
		err = t.GetTorrents(s)
		if err == nil {
			break
		}

		s.log.Warnf("failed to get torrents from torrenter: %q", err)
	}
	return err
}

// Notify sends a notification
func (s *ShowEpisode) Notify() error {
	var err error
	for _, n := range s.Notifiers {
		err = n.Notify(s)
		if err == nil {
			break
		}

		s.log.Warnf("failed to send a notification from notifier: %q", err)
	}
	return err
}

//  create the show nfo if it doesn't exists yet
func (s *ShowEpisode) storePath() string {
	return filepath.Join(s.Dir, s.ShowTitle, fmt.Sprintf("Season %d", s.Season))
}

// createShowDir creates a new show and stores it
func (s *ShowEpisode) createShowDir() error {
	show := &Show{
		Title:     s.ShowTitle,
		ShowTitle: s.ShowTitle,
		TvdbID:    s.ShowTvdbID,
		ImdbID:    s.ShowImdbID,
		ShowConfig: ShowConfig{
			Dir:        s.Dir,
			Detailers:  s.Detailers,
			Notifiers:  s.Notifiers,
			Subtitlers: s.Subtitlers,
			Torrenters: s.Torrenters,
		},
		log: s.log,
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
	if path.Dir(s.Path) == storePath {
		s.log.Debug("show episode already in the destination folder")
		return nil
	}

	// Move the episode into the folder
	newPath := filepath.Join(storePath, path.Base(s.Path))
	s.log.Debug("Moving episode to folder")
	s.log.Debugf("Old path: %q", s.Path)
	s.log.Debugf("New path: %q", newPath)
	if err := os.Rename(s.Path, newPath); err != nil {
		return err
	}

	// Set the new movie path
	s.Path = newPath

	return nil
}

// Store create the show episode nfo and download the images
func (s *ShowEpisode) Store() error {
	if s.Path == "" {
		return ErrMissingShowEpisodeFilePath
	}

	if s.Dir == "" {
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
	if err := MarshalInFile(s, s.NfoPath()); err != nil {
		return err
	}

	return nil
}

// GetSubtitle implements the subtitle interface
func (s *ShowEpisode) GetSubtitle() error {
	var err error
	var subtitle Subtitle
	for _, subtitler := range s.Subtitlers {
		subtitle, err = subtitler.GetShowSubtitle(s)
		if err == nil {
			break
		}

		s.log.Warnf("failed to get subtitles from subtitler: %q", err)
	}

	if err == nil {
		file, err := os.Create(s.SubtitlePath())
		if err != nil {
			return err
		}
		defer file.Close()
		defer subtitle.Close()

		if _, err := io.Copy(file, subtitle); err != nil {
			return err
		}
	}

	return err
}
