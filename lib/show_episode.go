package polochon

import (
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
	Explorers  []Explorer
	Searchers  []Searcher
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

// GetSubtitles implements the subtitle interface
// If there is an error, it will be of type *errors.Collector
func (s *ShowEpisode) GetSubtitles(languages []Language, log *logrus.Entry) error {
	c := errors.NewCollector()

	var subtitles map[Language]Subtitle
	// We're going to ask subtitles in each language for each subtitles
	for _, lang := range languages {
		subtitlerLog := log.WithField("lang", lang)
		// Ask all the subtitlers
		for _, subtitler := range s.Subtitlers {
			subtitlerLog = subtitlerLog.WithField("subtitler", subtitler.Name())
			subtitle, err := subtitler.GetShowSubtitle(s, lang, subtitlerLog)
			if err == nil {
				// If there was no errors, add the subtitle to the map of
				// subtitles
				subtitles[lang] = subtitle
				break
			}

			c.Push(errors.Wrap(err).Ctx("Subtitler", subtitler.Name()))
		}
	}

	// If we found some subtitles, create the files and download the subtitle
	if len(subtitles) != 0 {
		for lang, subtitle := range subtitles {
			file, err := os.Create(s.SubtitlePath(lang))
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
	}

	if c.HasErrors() {
		return c
	}
	return nil
}
