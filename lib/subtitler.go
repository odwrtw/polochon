package polochon

import (
	"errors"
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
