package polochon

import "context"

// Subtitler all subtitler must implement it
type Subtitler interface {
	Module
	GetSubtitle(ctx context.Context, v any, lang Language) (*Subtitle, error)
	ListSubtitles(ctx context.Context, v any, lang Language) ([]*SubtitleEntry, error)
	DownloadSubtitle(ctx context.Context, v any, entry *SubtitleEntry) (*Subtitle, error)
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
