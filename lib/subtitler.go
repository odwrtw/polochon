package polochon

import (
	"errors"
	"fmt"
	"io"

	"github.com/Sirupsen/logrus"
)

// ErrNoSubtitleFound trigger when no subtitle found
var ErrNoSubtitleFound = errors.New("No subtitle found")

// Subtitler all subtitler must implement it
type Subtitler interface {
	Module
	GetShowSubtitle(*ShowEpisode, *logrus.Entry) (Subtitle, error)
	GetMovieSubtitle(*Movie, *logrus.Entry) (Subtitle, error)
}

// Subtitle represents a subtitle
type Subtitle interface {
	io.ReadCloser
}

// RegisterSubtitler helps register a new Subtitler
func RegisterSubtitler(name string, f func(params map[string]interface{}) (Subtitler, error)) {
	if _, ok := registeredModules.Subtitlers[name]; ok {
		panic(fmt.Sprintf("modules: %q of type %q is already registered", name, TypeDetailer))
	}

	// Register the module
	registeredModules.Subtitlers[name] = f
}
