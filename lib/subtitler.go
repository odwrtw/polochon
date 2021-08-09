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
	GetSubtitle(interface{}, Language, *logrus.Entry) (*Subtitle, error)
}

// Subtitle represents a subtitle
type Subtitle struct {
	File
	io.ReadCloser

	Lang  Language
	Video Video
}

// NewSubtitleFromVideo returns a subtitle from a video
func NewSubtitleFromVideo(v Video, l Language) *Subtitle {
	file := NewFile(v.GetFile().SubtitlePath(l))
	return &Subtitle{
		File:  *file,
		Lang:  l,
		Video: v,
	}
}

// NewSubtitleFromReadCloser returns a subtitle from a io.ReadCloser
func NewSubtitleFromReadCloser(rc io.ReadCloser) *Subtitle {
	return &Subtitle{
		ReadCloser: rc,
	}
}

// Subtitlable represents a ressource which can be subtitled
type Subtitlable interface {
	SubtitlePath(Language) string
	GetSubtitlers() []Subtitler
}
