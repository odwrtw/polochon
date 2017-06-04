package library

import (
	"io"
	"os"

	"github.com/odwrtw/errors"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/media_index"
	"github.com/sirupsen/logrus"
)

// Custom errors
var (
	ErrInvalidIndexVideoType      = errors.New("library: invalid index video type")
	ErrMissingMovieFilePath       = errors.New("library: movie has no file path")
	ErrMissingMovieImageURL       = errors.New("library: missing movie images URL")
	ErrMissingShowImageURL        = errors.New("library: missing URL to download show images")
	ErrMissingShowEpisodeFilePath = errors.New("library: missing file path")
)

// Config represents configuration for the library
type Config struct {
	MovieDir string
	ShowDir  string
}

// Library represents a collection of videos
type Library struct {
	configuration.LibraryConfig
	movieIndex        *index.MovieIndex
	showIndex         *index.ShowIndex
	showConfig        polochon.ShowConfig
	movieConfig       polochon.MovieConfig
	fileConfig        polochon.FileConfig
	SubtitleLanguages []polochon.Language
}

// New returns a list of videos
func New(config *configuration.Config) *Library {
	return &Library{
		movieIndex:        index.NewMovieIndex(),
		showIndex:         index.NewShowIndex(),
		showConfig:        config.Show,
		movieConfig:       config.Movie,
		fileConfig:        config.File,
		SubtitleLanguages: config.SubtitleLanguages,
		LibraryConfig:     config.Library,
	}
}

// HasVideo checks if the video is in the library
func (l *Library) HasVideo(video polochon.Video) (bool, error) {
	switch v := video.(type) {
	case *polochon.Movie:
		return l.HasMovie(v.ImdbID)
	case *polochon.ShowEpisode:
		return l.HasShowEpisode(v.ShowImdbID, v.Season, v.Episode)
	default:
		return false, ErrInvalidIndexVideoType
	}
}

// Add video
func (l *Library) Add(video polochon.Video, log *logrus.Entry) error {
	switch v := video.(type) {
	case *polochon.Movie:
		return l.AddMovie(v, log)
	case *polochon.ShowEpisode:
		return l.AddShowEpisode(v, log)
	default:
		return ErrInvalidIndexVideoType
	}
}

// Delete will delete the video
func (l *Library) Delete(video polochon.Video, log *logrus.Entry) error {
	switch v := video.(type) {
	case *polochon.Movie:
		return l.DeleteMovie(v, log)
	case *polochon.ShowEpisode:
		return l.DeleteShowEpisode(v, log)
	default:
		return ErrInvalidIndexVideoType
	}
}

// AddSubtitles gets and downloads subtitles of different languages
func (l *Library) AddSubtitles(video polochon.Subtitlable, languages []polochon.Language, log *logrus.Entry) ([]polochon.Language, error) {
	c := errors.NewCollector()
	addedSubtitles := []polochon.Language{}

	// We're going to ask subtitles in each language for each subtitles
	for _, lang := range languages {
		subtitlerLog := log.WithField("lang", lang)
		// Ask all the subtitlers
		for _, subtitler := range video.GetSubtitlers() {
			subtitlerLog = subtitlerLog.WithField("subtitler", subtitler.Name())
			subtitle, err := subtitler.GetSubtitle(video, lang, subtitlerLog)
			if err != nil {
				// If there was no errors, add the subtitle to the map of
				// subtitles
				c.Push(errors.Wrap(err).Ctx("Subtitler", subtitler.Name()).Ctx("lang", lang))
				continue
			}

			err = l.DownloadSubtitle(subtitle, video, lang)
			if err != nil {
				c.Push(errors.Wrap(err).Ctx("Subtitler", subtitler.Name()).Ctx("lang", lang))
				continue
			}
			err = l.AddSubtitleIndex(video, lang)
			if err != nil {
				c.Push(errors.Wrap(err).Ctx("Subtitler", subtitler.Name()).Ctx("lang", lang))
				continue
			}
			addedSubtitles = append(addedSubtitles, lang)
			break
		}
	}
	if c.HasErrors() {
		if c.IsFatal() {
			return nil, c
		}
		log.Warnf("Got non fatal errors while getting subtitles: %s", c)
	}

	return addedSubtitles, nil
}

// DownloadSubtitle will download the subtitle
func (l *Library) DownloadSubtitle(subtitle io.ReadCloser, v polochon.Subtitlable, lang polochon.Language) error {
	file, err := os.Create(v.SubtitlePath(lang))
	if err != nil {
		return err

	}
	defer file.Close()
	defer subtitle.Close()

	if _, err := io.Copy(file, subtitle); err != nil {
		return err
	}
	return nil
}

// AddSubtitleIndex will add a subtitle in the index
func (l *Library) AddSubtitleIndex(video polochon.Subtitlable, lang polochon.Language) error {
	switch v := video.(type) {
	case *polochon.Movie:
		ok, err := l.movieIndex.HasSubtitle(v.ImdbID, lang)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		return l.movieIndex.AddSubtitle(v, lang)
	case *polochon.ShowEpisode:
		ok, err := l.showIndex.HasEpisodeSubtitle(v.ShowImdbID, v.Season, v.Episode, lang)
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		return l.showIndex.AddSubtitle(v, lang)
	default:
		return ErrInvalidIndexVideoType
	}
}
