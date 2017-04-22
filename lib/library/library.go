package library

import (
	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/errors"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/media_index"
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
	Config
	movieIndex  *index.MovieIndex
	showIndex   *index.ShowIndex
	showConfig  polochon.ShowConfig
	movieConfig polochon.MovieConfig
	fileConfig  polochon.FileConfig
}

// New returns a list of videos
func New(fileConfig polochon.FileConfig, movieConfig polochon.MovieConfig, showConfig polochon.ShowConfig, vsConfig Config) *Library {
	return &Library{
		movieIndex:  index.NewMovieIndex(),
		showIndex:   index.NewShowIndex(),
		showConfig:  showConfig,
		movieConfig: movieConfig,
		fileConfig:  fileConfig,
		Config:      vsConfig,
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
