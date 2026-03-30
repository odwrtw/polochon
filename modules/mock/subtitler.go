package mock

import (
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// GetSubtitle implements the Subtitler interface
func (mock *Mock) GetSubtitle(v any, lang polochon.Language, log *logrus.Entry) (*polochon.Subtitle, error) {
	video, ok := v.(polochon.Video)
	if !ok {
		return nil, ErrInvalidArgument
	}

	sub := polochon.NewSubtitleFromVideo(video, lang)
	sub.Data = []byte("subtitle in " + string(lang))

	return sub, nil
}

// ListSubtitles implements the Subtitler interface
func (mock *Mock) ListSubtitles(v any, lang polochon.Language, log *logrus.Entry) ([]*polochon.SubtitleEntry, error) {
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
func (mock *Mock) DownloadSubtitle(v any, entry *polochon.SubtitleEntry, log *logrus.Entry) (*polochon.Subtitle, error) {
	video, ok := v.(polochon.Video)
	if !ok {
		return nil, ErrInvalidArgument
	}

	sub := polochon.NewSubtitleFromVideo(video, entry.Language)
	sub.Data = []byte("downloaded subtitle for language: " + string(entry.Language))
	return sub, nil
}
