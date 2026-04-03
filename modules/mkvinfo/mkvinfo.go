package mkvinfo

import (
	"context"
	"fmt"
	"log/slog"

	polochon "github.com/odwrtw/polochon/lib"
)

var _ polochon.Guesser = (*MKVInfo)(nil)
var _ polochon.Subtitler = (*MKVInfo)(nil)

// Register guessit as a Guesser
func init() {
	polochon.RegisterModule(&MKVInfo{})
}

// Module constants
const (
	moduleName = "mkvinfo"
)

// MKVInfo is used to parse mkv tracks
type MKVInfo struct {
}

// Init implements the module interface
func (m *MKVInfo) Init(p []byte, _ *slog.Logger) error {
	return nil
}

// Name implements the Module interface
func (m *MKVInfo) Name() string {
	return moduleName
}

// Status implements the Module interface
func (m *MKVInfo) Status() (polochon.ModuleStatus, error) {
	return polochon.StatusOK, nil
}

// GuessMetadata implements the Guesser interface
func (m *MKVInfo) GuessMetadata(file *polochon.File) (*polochon.VideoMetadata, error) {
	entries, err := ParseFile(file)
	if err != nil {
		return nil, err
	}

	return Metadata(entries), nil
}

// Guess implements the Guesser interface
func (m *MKVInfo) Guess(file polochon.File, movieConf polochon.MovieConfig, showConf polochon.ShowConfig) (polochon.Video, error) {
	return nil, polochon.ErrNotAvailable
}

// ListSubtitles implements the Subtitler interface.
func (m *MKVInfo) ListSubtitles(_ context.Context, v any, lang polochon.Language) ([]*polochon.SubtitleEntry, error) {
	video, ok := v.(polochon.Video)
	if !ok {
		return nil, ErrNotAVideo
	}

	entries, err := ParseFile(video.GetFile())
	if err != nil {
		return nil, err
	}

	if !HasSubtitle(entries, lang) {
		return nil, polochon.ErrNoSubtitleFound
	}

	return []*polochon.SubtitleEntry{
		{
			Language:    lang,
			Embedded:    true,
			ID:          string(lang),
			Description: fmt.Sprintf("Embedded %s subtitle", lang),
		},
	}, nil
}

// DownloadSubtitle implements the Subtitler interface.
func (m *MKVInfo) DownloadSubtitle(_ context.Context, v any, entry *polochon.SubtitleEntry) (*polochon.Subtitle, error) {
	video, ok := v.(polochon.Video)
	if !ok {
		return nil, ErrNotAVideo
	}
	return &polochon.Subtitle{
		Embedded: true,
		Lang:     entry.Language,
		Video:    video,
	}, nil
}

// GetSubtitle implements the Subtitler interface
func (m *MKVInfo) GetSubtitle(_ context.Context, v any, lang polochon.Language) (*polochon.Subtitle, error) {
	video, ok := v.(polochon.Video)
	if !ok {
		return nil, ErrNotAVideo
	}

	entries, err := ParseFile(video.GetFile())
	if err != nil {
		return nil, err
	}

	if !HasSubtitle(entries, lang) {
		return nil, polochon.ErrNoSubtitleFound
	}

	return &polochon.Subtitle{
		Embedded: true,
		Lang:     lang,
		Video:    video,
	}, nil
}
