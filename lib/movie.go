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

// Movie errors
var (
	ErrMissingMovieFilePath = errors.New("polochon: movie has no file path")
	ErrMissingMovieImageURL = errors.New("polochon: missing movie images URL")
	ErrMissingMovieDir      = errors.New("polochon: missing movie dir in config")
)

// Movie represents a movie
type Movie struct {
	XMLName       xml.Name  `xml:"movie" json:"-"`
	ImdbID        string    `xml:"id" json:"imdb_id"`
	OriginalTitle string    `xml:"originaltitle" json:"original_title"`
	Plot          string    `xml:"plot" json:"plot"`
	Rating        float32   `xml:"rating" json:"rating"`
	Runtime       int       `xml:"runtime" json:"runtime"`
	SortTitle     string    `xml:"sorttitle" json:"sort_title"`
	Tagline       string    `xml:"tagline" json:"tag_line"`
	Thumb         string    `xml:"thumb" json:"thumb"`
	Fanart        string    `xml:"customfanart" json:"fanart"`
	Title         string    `xml:"title" json:"title"`
	TmdbID        int       `xml:"tmdbid" json:"tmdb_id"`
	Votes         int       `xml:"votes" json:"votes"`
	Year          int       `xml:"year" json:"year"`
	Torrents      []Torrent `xml:"-" json:"-"`
	File          *File     `xml:"-" json:"file"`
	log           *logrus.Entry
	config        *MovieConfig
}

// NewMovie returns a new movie
func NewMovie() *Movie {
	return &Movie{
		XMLName: xml.Name{Space: "", Local: "movie"},
	}
}

// Type implements the video interface
func (m *Movie) Type() VideoType {
	return MovieType
}

// SetFile implements the video interface
func (m *Movie) SetFile(f *File) {
	m.File = f
}

// SetConfig implements the video interface
func (m *Movie) SetConfig(c *Config) {
	// Only keep movie config from global config
	m.config = &c.Movie

	// Set logger
	m.log = c.Log.WithFields(logrus.Fields{
		"type": "movie",
	})
}

// readShowSeasonNFO deserialized a XML file into a ShowSeason
func readMovieNFO(r io.Reader) (*Movie, error) {
	m := &Movie{}

	if err := readNFO(r, m); err != nil {
		return nil, err
	}

	return m, nil
}

// GetDetails helps getting infos for a movie
func (m *Movie) GetDetails() error {
	var err error
	for _, d := range m.config.Detailers {
		err = d.GetDetails(m)
		if err == nil {
			break
		}
		m.log.Warnf("failed to get details from detailer: %q", err)
	}
	return err
}

// GetTorrents helps getting the torrent files for a movie
func (m *Movie) GetTorrents() error {
	var err error
	for _, t := range m.config.Torrenters {
		err = t.GetTorrents(m)
		if err == nil {
			break
		}

		m.log.Warnf("failed to get torrents from torrenter: %q", err)
	}
	return err
}

// Notify sends a notification
func (m *Movie) Notify() error {
	var err error
	for _, n := range m.config.Notifiers {
		err = n.Notify(m)
		if err == nil {
			break
		}

		m.log.Warnf("failed to send a notification from notifier: %q", err)
	}
	return err
}

// storePath returns the movie store path from the config
func (m *Movie) storePath() string {
	if m.Year != 0 {
		return filepath.Join(m.config.Dir, fmt.Sprintf("%s (%d)", m.Title, m.Year))
	}

	return filepath.Join(m.config.Dir, m.Title)
}

// move helps move the movie to the expected destination
func (m *Movie) move() error {
	storePath := m.storePath()

	// If the movie already in the right dir there is nothing to do
	if path.Dir(m.File.Path) == storePath {
		m.log.Debug("movie already in the destination folder")
		return nil
	}

	// Remove movie dir if it exisits
	if _, err := os.Stat(storePath); err == nil {
		m.log.Debug("Movie folder exists, remove it")
		if err = os.RemoveAll(storePath); err != nil {
			return err
		}
	}

	// Create the folder
	if err := os.Mkdir(storePath, os.ModePerm); err != nil {
		return err
	}

	// Move the movie into the folder
	newPath := filepath.Join(storePath, path.Base(m.File.Path))
	m.log.Debugf("Moving movie %q to folder", m.Title)
	m.log.Debugf("Old path: %q", m.File.Path)
	m.log.Debugf("New path: %q", newPath)
	if err := os.Rename(m.File.Path, newPath); err != nil {
		return err
	}

	// Set the new movie path
	m.File.Path = newPath

	return nil
}

// Store stores the movie according to the config
func (m *Movie) Store() error {
	if m.File == nil {
		return ErrMissingMovieFilePath
	}

	if m.config == nil || m.config.Dir == "" {
		return ErrMissingMovieDir
	}

	// Local logs
	m.log = m.log.WithFields(logrus.Fields{
		"function": "store",
		"title":    m.Title,
	})

	// Move the file
	if err := m.move(); err != nil {
		return err
	}

	// Write NFO into the file
	if err := MarshalInFile(m, m.File.NfoPath()); err != nil {
		return err
	}

	// Download images
	if err := m.downloadImages(); err != nil {
		return err
	}

	return nil
}

// Function to be overwritten during the tests
var downloadMovieImage = func(URL, savePath string, log *logrus.Entry) error {
	return download(URL, savePath, log)
}

func (m *Movie) downloadImages() error {
	if m.Fanart == "" || m.Thumb == "" {
		return ErrMissingMovieImageURL
	}

	// Download images
	images := map[string]string{
		m.Fanart: m.File.MovieFanartPath(),
		m.Thumb:  filepath.Join(path.Dir(m.File.Path), "/poster.jpg"),
	}
	for URL, savePath := range images {
		if err := downloadMovieImage(URL, savePath, m.log); err != nil {
			return err
		}
	}

	return nil
}

// GetSubtitle implements the subtitle interface
func (m *Movie) GetSubtitle() error {

	return nil
}
