package mock

import (
	"context"

	polochon "github.com/odwrtw/polochon/lib"
)

// GetSubtitle implements the Subtitler interface
func (mock *Mock) GetSubtitle(_ context.Context, v any, lang polochon.Language) (*polochon.Subtitle, error) {
	video, ok := v.(polochon.Video)
	if !ok {
		return nil, ErrInvalidArgument
	}

	sub := polochon.NewSubtitleFromVideo(video, lang)
	sub.Data = []byte("subtitle in " + string(lang))

	return sub, nil
}

// ListSubtitles implements the Subtitler interface
func (mock *Mock) ListSubtitles(_ context.Context, v any, lang polochon.Language) ([]*polochon.SubtitleEntry, error) {
	if _, ok := v.(polochon.Video); !ok {
		return nil, ErrInvalidArgument
	}

	return []*polochon.SubtitleEntry{
		{
			Language:    lang,
			ID:          "mock-id",
			Description: "mock subtitle",
		},
	}, nil
}

// DownloadSubtitle implements the Subtitler interface
func (mock *Mock) DownloadSubtitle(_ context.Context, v any, entry *polochon.SubtitleEntry) (*polochon.Subtitle, error) {
	video, ok := v.(polochon.Video)
	if !ok {
		return nil, ErrInvalidArgument
	}

	sub := polochon.NewSubtitleFromVideo(video, entry.Language)
	sub.Data = []byte("downloaded subtitle for language: " + string(entry.Language))
	return sub, nil
}
