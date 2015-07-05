package polochon

import (
	"errors"
	"log"

	"github.com/Sirupsen/logrus"
)

// ErrNoSubtitleFound trigger when no subtitle found
var ErrNoSubtitleFound = errors.New("No subtitle found")

// Subtitiler all subtitiler must implement it
type Subtitiler interface {
	GetShowSubtitle(*ShowEpisode) (Subtitle, error)
	GetMovieSubtitle(*Movie) (Subtitle, error)
}

// Subtitle represents a subtitle
type Subtitle interface {
	Read([]byte) (int, error)
	Close()
}

// RegisterSubtitiler helps register a new Subtitiler
func RegisterSubtitiler(name string, f func(params map[string]interface{}, log *logrus.Entry) (Subtitiler, error)) {
	if _, ok := registeredModules.Subtitilers[name]; ok {
		log.Panicf("modules: %q of type %q is already registered", name, TypeSubtitiler)
	}

	// Register the module
	registeredModules.Subtitilers[name] = f
}
