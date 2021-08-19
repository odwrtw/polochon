package library

import (
	"errors"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	index "github.com/odwrtw/polochon/lib/media_index"
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

// Library represents a collection of videos
type Library struct {
	configuration.LibraryConfig
	movieIndex        *index.MovieIndex
	showIndex         *index.ShowIndex
	showConfig        polochon.ShowConfig
	movieConfig       polochon.MovieConfig
	fileConfig        polochon.FileConfig
	downloaderConfig  configuration.DownloaderConfig
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
		downloaderConfig:  config.Downloader,
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

// Add adds a video in the library
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

// Delete deletes a video from the library
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
