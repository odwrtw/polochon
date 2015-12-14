package polochon

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"

	"github.com/Sirupsen/logrus"
)

// ShowConfig represents the configuration for a show and its show episodes
type ShowConfig struct {
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
	ReleaseGroup  string    `xml:"-"`
	Torrents      []Torrent `xml:"-" json:"torrents"`
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

// GetFile implements the video interface
func (s *ShowEpisode) GetFile() *File {
	return &s.File
}

// SetFile implements the video interface
func (s *ShowEpisode) SetFile(f *File) {
	s.File = *f
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
