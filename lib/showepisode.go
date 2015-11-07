package polochon

import (
	"encoding/json"
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

// ShowConfig represents the configuration for a show and its show episodes
type ShowConfig struct {
	Dir        string
	Calendar   Calendar
	Detailers  []Detailer
	Notifiers  []Notifier
	Subtitlers []Subtitler
	Torrenters []Torrenter
}

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

// MarshalJSON is a custom marshal function to handle public path
func (s *ShowEpisode) MarshalJSON() ([]byte, error) {
	var aux struct {
		ShowEpisode
		Slug string `json:"slug"`
	}
	aux.Slug = s.Slug()
	aux.ShowEpisode = *s

	return json.Marshal(aux)
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

// NewShowEpisodeFromPath returns a new ShowEpisode object from path, it loads the nfo
func NewShowEpisodeFromPath(Sconf ShowConfig, Fconf FileConfig, log *logrus.Entry, path string) (*ShowEpisode, error) {
	file := NewFileWithConfig(path, Fconf)

	// Open the NFO
	nfoFile, err := os.Open(file.NfoPath())
	if err != nil {
		return nil, err
	}
	defer nfoFile.Close()

	// Unmarshal the NFO into an episode
	ep, err := readShowEpisodeNFO(nfoFile, Sconf)
	if err != nil {
		return nil, err
	}
	ep.SetFile(file)
	ep.SetLogger(log)
	return ep, nil
}

// Type implements the video interface
func (s *ShowEpisode) Type() VideoType {
	return ShowEpisodeType
}

// GetFile implements the video interface
func (s *ShowEpisode) GetFile() *File {
	return &s.File
}

// SetFile implements the video interface
func (s *ShowEpisode) SetFile(f *File) {
	s.Path = f.Path
	s.ExcludeFileContaining = f.ExcludeFileContaining
	s.VideoExtentions = f.VideoExtentions
	s.AllowedExtentionsToDelete = f.AllowedExtentionsToDelete
	s.Guesser = f.Guesser
}

// SetLogger sets the logger
func (s *ShowEpisode) SetLogger(log *logrus.Entry) {
	s.log = log.WithField("type", "show_episode")
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
		err = d.GetDetails(s, s.log)
		if err == nil {
			s.log.Debugf("got details from detailer: %q", d.Name())
			break
		}
		s.log.Warnf("failed to get details from detailer: %q: %q", d.Name(), err)
	}
	return err
}

// GetTorrents helps getting the torrent files for a movie
func (s *ShowEpisode) GetTorrents() error {
	var err error
	for _, t := range s.Torrenters {
		err = t.GetTorrents(s, s.log)
		if err == nil {
			break
		}
	}
	return err
}

// Notify sends a notification
func (s *ShowEpisode) Notify() error {
	var err error
	for _, n := range s.Notifiers {
		err = n.Notify(s, s.log)
		if err == nil {
			break
		}

		s.log.Warnf("failed to send a notification from notifier: %q: %q", n.Name(), err)
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
		subtitle, err = subtitler.GetShowSubtitle(s, s.log)
		if err == nil {
			s.log.Infof("Got subtitle from subtitiler %q", subtitler.Name())
			break
		}

		s.log.Warnf("failed to get subtitles from subtitiler %q: %q", subtitler.Name(), err)
	}

	if subtitle != nil {
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

// Slug will slug the show episode
func (s *ShowEpisode) Slug() string {
	return slug(fmt.Sprintf("%s-s%02de%02d", s.ShowTitle, s.Season, s.Episode))
}

// Delete implements the Video interface
func (s *ShowEpisode) Delete() error {
	s.log.Infof("Removing ShowEpisode %q", s.Path)
	// Remove the episode
	err := os.RemoveAll(s.Path)
	if err != nil {
		s.log.Warnf("Couldn't remove %q : %s", s.Path, err)
		return err
	}
	pathWithoutExt := s.filePathWithoutExt()

	// Remove also the .nfo and .srt files
	for _, ext := range []string{"nfo", "srt"} {
		fileToDelete := fmt.Sprintf("%s.%s", pathWithoutExt, ext)
		s.log.Debugf("Removing %q", fileToDelete)
		// Remove file
		err := os.RemoveAll(fileToDelete)
		if err != nil {
			s.log.Warnf("Couldn't remove %q : %s", fileToDelete, err)
			return err
		}
	}

	return nil
}

// DeleteSeason implements the Video interface
func (s *ShowEpisode) DeleteSeason() error {
	s.log.Infof("Removing Season %s", s.storePath())
	return os.RemoveAll(s.storePath())
}
