package polochon

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
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
	MovieConfig `xml:"-" json:"-"`
	File
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
	Torrents      []Torrent `xml:"-" json:"torrents"`
}

// MarshalJSON is a custom marshal function to handle public path
func (m *Movie) MarshalJSON() ([]byte, error) {
	var aux struct {
		Movie
		Slug string `json:"slug"`
	}
	aux.Slug = m.Slug()
	aux.Movie = *m

	return json.Marshal(aux)
}

// NewMovie returns a new movie
func NewMovie(movieConfig MovieConfig) *Movie {
	return &Movie{
		MovieConfig: movieConfig,
		XMLName:     xml.Name{Space: "", Local: "movie"},
	}
}

// NewMovieFromFile returns a new movie from a file
func NewMovieFromFile(movieConfig MovieConfig, file File) *Movie {
	return &Movie{
		MovieConfig: movieConfig,
		File:        file,
		XMLName:     xml.Name{Space: "", Local: "movie"},
	}
}

// NewMovieFromPath returns a new movie object from path, it loads the nfo
func NewMovieFromPath(Mconf MovieConfig, Fconf FileConfig, path string) (*Movie, error) {
	file := NewFileWithConfig(path, Fconf)

	// Open the NFO
	nfoFile, err := os.Open(file.NfoPath())
	if err != nil {
		return nil, err
	}
	defer nfoFile.Close()

	// Unmarshal the NFO into an episode
	movie, err := readMovieNFO(nfoFile, Mconf)
	if err != nil {
		return nil, err
	}
	movie.SetFile(file)
	return movie, nil
}

// SetFile implements the video interface
func (m *Movie) SetFile(f *File) {
	m.File = *f
}

// GetFile implements the video interface
func (m *Movie) GetFile() *File {
	return &m.File
}

// readShowSeasonNFO deserialized a XML file into a ShowSeason
func readMovieNFO(r io.Reader, conf MovieConfig) (*Movie, error) {
	m := NewMovie(conf)

	if err := readNFO(r, m); err != nil {
		return nil, err
	}

	return m, nil
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
		err := d.GetDetails(m, log)
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
		err := t.GetTorrents(m, log)
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
		subtitle, err = subtitler.GetMovieSubtitle(m, log)
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

// Slug will slug the movie name
func (m *Movie) Slug() string {
	return slug(fmt.Sprintf("%s", m.Title))
}
