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
			Language: lang,
			Release:  "mock.release." + string(lang),
		},
	}, nil
}
