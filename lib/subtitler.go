package polochon

import (
	"errors"
	"io"
	"log"

	"github.com/Sirupsen/logrus"
)

// ErrNoSubtitleFound trigger when no subtitle found
var ErrNoSubtitleFound = errors.New("No subtitle found")

// Subtitler all subtitler must implement it
type Subtitler interface {
	Module
	GetShowSubtitle(*ShowEpisode) (Subtitle, error)
	GetMovieSubtitle(*Movie) (Subtitle, error)
}

// Subtitle represents a subtitle
type Subtitle interface {
	io.ReadCloser
}

// RegisterSubtitler helps register a new Subtitler
func RegisterSubtitler(name string, f func(params map[string]interface{}, log *logrus.Entry) (Subtitler, error)) {
	if _, ok := registeredModules.Subtitlers[name]; ok {
		log.Panicf("modules: %q of type %q is already registered", name, TypeSubtitler)
	}

	// Register the module
	registeredModules.Subtitlers[name] = f
}
