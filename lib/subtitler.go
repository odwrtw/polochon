package polochon

import "github.com/sirupsen/logrus"

// Subtitler all subtitler must implement it
type Subtitler interface {
	Module
	GetSubtitle(any, Language, *logrus.Entry) (*Subtitle, error)
	ListSubtitles(any, Language, *logrus.Entry) ([]*SubtitleEntry, error)
	DownloadSubtitle(any, *SubtitleEntry, *logrus.Entry) (*Subtitle, error)
}

// FindSubtitler returns the first subtitler with the given name, or nil if not found.
func FindSubtitler(subtitlers []Subtitler, name string) Subtitler {
	for _, s := range subtitlers {
		if s.Name() == name {
			return s
		}
	}
	return nil
}

// Subtitlable represents a ressource which can be subtitled
type Subtitlable interface {
	SubtitlePath(Language) string
	GetSubtitlers() []Subtitler
}
