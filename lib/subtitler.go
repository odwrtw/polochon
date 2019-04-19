package polochon

import (
	"errors"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
)

// ErrNoSubtitleFound trigger when no subtitle found
var ErrNoSubtitleFound = errors.New("polochon: no subtitle found")

// Subtitler all subtitler must implement it
type Subtitler interface {
	Module
	GetSubtitle(interface{}, Language, *logrus.Entry) (Subtitle, error)
}

// Subtitle represents a subtitle
type Subtitle interface {
	io.ReadCloser
}

// Subtitlable represents a ressource which can be subtitled
type Subtitlable interface {
	SubtitlePath(Language) string
	GetSubtitlers() []Subtitler
}

// RegisterSubtitler helps register a new Subtitler
func RegisterSubtitler(name string, f func(params []byte) (Subtitler, error)) {
	if _, ok := registeredModules.Subtitlers[name]; ok {
		panic(fmt.Sprintf("modules: %q of type %q is already registered", name, TypeSubtitler))
	}

	// Register the module
	registeredModules.Subtitlers[name] = f
}
