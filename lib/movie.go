package polochon

import (
	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/errors"
)

// MovieConfig represents the configuration for a movie
type MovieConfig struct {
	Torrenters []Torrenter
	Detailers  []Detailer
	Subtitlers []Subtitler
	Explorers  []Explorer
	Searchers  []Searcher
}

// Movie represents a movie
type Movie struct {
	MovieConfig `json:"-"`
	File
	ImdbID        string    `json:"imdb_id"`
	OriginalTitle string    `json:"original_title"`
	Plot          string    `json:"plot"`
	Rating        float32   `json:"rating"`
	Runtime       int       `json:"runtime"`
	SortTitle     string    `json:"sort_title"`
	Tagline       string    `json:"tag_line"`
	Thumb         string    `json:"thumb"`
	Fanart        string    `json:"fanart"`
	Title         string    `json:"title"`
	TmdbID        int       `json:"tmdb_id"`
	Votes         int       `json:"votes"`
	Year          int       `json:"year"`
	Genres        []string  `json:"genres"`
	Torrents      []Torrent `json:"torrents"`
}

// NewMovie returns a new movie
func NewMovie(movieConfig MovieConfig) *Movie {
	return &Movie{
		MovieConfig: movieConfig,
	}
}

// NewMovieFromFile returns a new movie from a file
func NewMovieFromFile(movieConfig MovieConfig, file File) *Movie {
	return &Movie{
		MovieConfig: movieConfig,
		File:        file,
	}
}

// GetTorrenters implements the Torrentable interface
func (m *Movie) GetTorrenters() []Torrenter {
	return m.Torrenters
}

// GetSubtitlers implements the Subtitlable interface
func (m *Movie) GetSubtitlers() []Subtitler {
	return m.Subtitlers
}

// GetDetailers implements the Detailable interface
func (m *Movie) GetDetailers() []Detailer {
	return m.Detailers
}

// SetFile implements the video interface
func (m *Movie) SetFile(f *File) {
	m.File = *f
}

// GetFile implements the video interface
func (m *Movie) GetFile() *File {
	return &m.File
}

// GetTorrents helps getting the torrent files for a movie
// If there is an error, it will be of type *errors.Collector
func (m *Movie) GetTorrents(log *logrus.Entry) error {
	c := errors.NewCollector()

	for _, t := range m.Torrenters {
		torrenterLog := log.WithField("torrenter", t.Name())
		err := t.GetTorrents(m, torrenterLog)
		if err == nil {
			break
		}
		c.Push(errors.Wrap(err).Ctx("Torrenter", t.Name()))
	}

	if c.HasErrors() {
		return c
	}
	return nil
}
