package polochon

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/errors"
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
	ShowConfig `json:"-"`
	File
	Title         string    `json:"title"`
	ShowTitle     string    `json:"-"`
	Season        int       `json:"season"`
	Episode       int       `json:"episode"`
	TvdbID        int       `json:"tvdb_id"`
	Aired         string    `json:"aired"`
	Plot          string    `json:"plot"`
	Runtime       int       `json:"runtime"`
	Thumb         string    `json:"thumb"`
	Rating        float32   `json:"rating"`
	ShowImdbID    string    `json:"show_imdb_id"`
	ShowTvdbID    int       `json:"show_tvdb_id"`
	EpisodeImdbID string    `json:"imdb_id"`
	ReleaseGroup  string    `json:"release_group"`
	Torrents      []Torrent `json:"torrents"`
	Show          *Show     `json:"-"`
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
	}
}

// NewShowEpisodeFromFile returns a new show episode from a file
func NewShowEpisodeFromFile(showConf ShowConfig, file File) *ShowEpisode {
	return &ShowEpisode{
		ShowConfig: showConf,
		File:       file,
	}
}

// GetFile implements the video interface
func (s *ShowEpisode) GetFile() *File {
	return &s.File
}

// SetFile implements the video interface
func (s *ShowEpisode) SetFile(f *File) {
	s.File = *f
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
		detailerLog := log.WithField("detailer", d.Name())
		err := d.GetDetails(s, detailerLog)
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
		torrenterLog := log.WithField("torrenter", t.Name())
		err := t.GetTorrents(s, torrenterLog)
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
		subtitlerLog := log.WithField("subtitler", subtitler.Name())
		subtitle, err = subtitler.GetShowSubtitle(s, subtitlerLog)
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
