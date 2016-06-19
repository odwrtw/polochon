package polochon

import (
	"io"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/errors"
)

// MovieConfig represents the configuration for a movie
type MovieConfig struct {
	Torrenters []Torrenter
	Detailers  []Detailer
	Subtitlers []Subtitler
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

// SetFile implements the video interface
func (m *Movie) SetFile(f *File) {
	m.File = *f
}

// GetFile implements the video interface
func (m *Movie) GetFile() *File {
	return &m.File
}

// GetDetails helps getting infos for a movie
// If there is an error, it will be of type *errors.Collector
func (m *Movie) GetDetails(log *logrus.Entry) error {
	c := errors.NewCollector()

	if len(m.Detailers) == 0 {
		c.Push(errors.Wrap("No detailer available").Fatal())
		return c
	}

	var done bool
	for _, d := range m.Detailers {
		detailerLog := log.WithField("detailer", d.Name())
		err := d.GetDetails(m, detailerLog)
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

// GetSubtitle implements the subtitle interface
// If there is an error, it will be of type *errors.Collector
func (m *Movie) GetSubtitle(log *logrus.Entry) error {
	c := errors.NewCollector()

	var subtitle Subtitle
	for _, subtitler := range m.Subtitlers {
		var err error
		subtitlerLog := log.WithField("subtitler", subtitler.Name())
		subtitle, err = subtitler.GetMovieSubtitle(m, subtitlerLog)
		if err == nil {
			break
		}

		c.Push(errors.Wrap(err).Ctx("Subtitler", subtitler.Name()))
	}

	if subtitle != nil {
		file, err := os.Create(m.File.SubtitlePath())
		if err != nil {
			c.Push(errors.Wrap(err).Fatal())
			return c
		}
		defer file.Close()
		defer subtitle.Close()

		if _, err := io.Copy(file, subtitle); err != nil {
			c.Push(errors.Wrap(err).Fatal())
			return c
		}
	}

	if c.HasErrors() {
		return c
	}
	return nil
}
