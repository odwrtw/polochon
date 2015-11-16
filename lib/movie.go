package polochon

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"

	"github.com/Sirupsen/logrus"
)

// MovieConfig represents the configuration for a movie
type MovieConfig struct {
	Torrenters []Torrenter
	Detailers  []Detailer
	Subtitlers []Subtitler
	Notifiers  []Notifier
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
	log           *logrus.Entry
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
func NewMovieFromPath(Mconf MovieConfig, Fconf FileConfig, log *logrus.Entry, path string) (*Movie, error) {
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
	movie.SetLogger(log)
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

// SetLogger sets the logger
func (m *Movie) SetLogger(log *logrus.Entry) {
	m.log = log.WithField("type", "movie")
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
func (m *Movie) GetDetails() error {
	var err error
	for _, d := range m.Detailers {
		m.log.Infof("Getting details from %q", d.Name())
		err = d.GetDetails(m, m.log)
		if err == nil {
			m.log.Debugf("got details from detailer: %q", d.Name())
			break
		}
		m.log.Warnf("failed to get details from detailer: %q", err)
	}
	return err
}

// GetTorrents helps getting the torrent files for a movie
func (m *Movie) GetTorrents() error {
	var err error
	for _, t := range m.Torrenters {
		m.log.Infof("Getting torrents from %q", t.Name())
		err = t.GetTorrents(m, m.log)
		if err == nil {
			break
		}
	}
	return err
}

// Notify sends a notification
func (m *Movie) Notify() error {
	var err error
	for _, n := range m.Notifiers {
		err = n.Notify(m, m.log)
		if err == nil {
			break
		}

		m.log.Warnf("failed to send a notification from notifier: %q", err)
	}
	return err
}

// GetSubtitle implements the subtitle interface
func (m *Movie) GetSubtitle() error {
	var err error
	var subtitle Subtitle
	for _, subtitler := range m.Subtitlers {
		subtitle, err = subtitler.GetMovieSubtitle(m, m.log)
		if err == nil {
			m.log.Infof("Got subtitle from subtitiler %q", subtitler.Name())
			break
		}

		m.log.Warnf("failed to get subtitles from subtitiler %q: %q", subtitler.Name(), err)
	}

	if subtitle != nil {
		file, err := os.Create(m.File.SubtitlePath())
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

// Slug will slug the movie name
func (m *Movie) Slug() string {
	return slug(fmt.Sprintf("%s", m.Title))
}
