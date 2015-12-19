package polochon

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/errors"
)

// ShowConfig represents the configuration for a show and its show episodes
type ShowConfig struct {
	Calendar   Calendar
	Detailers  []Detailer
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
	ReleaseGroup  string    `xml:"-"`
	Torrents      []Torrent `xml:"-" json:"torrents"`
	Show          *Show     `xml:"-" json:"-"`
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
func NewShowEpisodeFromPath(Sconf ShowConfig, Fconf FileConfig, path string) (*ShowEpisode, error) {
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
	return ep, nil
}

// GetFile implements the video interface
func (s *ShowEpisode) GetFile() *File {
	return &s.File
}

// SetFile implements the video interface
func (s *ShowEpisode) SetFile(f *File) {
	s.File = *f
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
// If there is an error, it will be of type *errors.Collector
func (s *ShowEpisode) GetDetails(log *logrus.Entry) error {
	c := errors.NewCollector()

	if len(s.Detailers) == 0 {
		c.Push(errors.Wrap("No detailer available").Fatal())
		return c
	}

	var done bool
	for _, d := range s.Detailers {
		err := d.GetDetails(s, log)
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
func (s *ShowEpisode) GetTorrents(log *logrus.Entry) error {
	c := errors.NewCollector()

	for _, t := range s.Torrenters {
		err := t.GetTorrents(s, log)
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
func (s *ShowEpisode) GetSubtitle(log *logrus.Entry) error {
	c := errors.NewCollector()

	var subtitle Subtitle
	for _, subtitler := range s.Subtitlers {
		var err error
		subtitle, err = subtitler.GetShowSubtitle(s, log)
		if err == nil {
			break
		}

		c.Push(errors.Wrap(err).Ctx("Subtitler", subtitler.Name()))
	}

	if subtitle != nil {
		file, err := os.Create(s.SubtitlePath())
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

// Slug will slug the show episode
func (s *ShowEpisode) Slug() string {
	return slug(fmt.Sprintf("%s-s%02de%02d", s.ShowTitle, s.Season, s.Episode))
}
