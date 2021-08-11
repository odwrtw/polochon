package mock

import (
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// GetSubtitle implements the Detailer interface
func (mock *Mock) GetSubtitle(v interface{}, lang polochon.Language, log *logrus.Entry) (*polochon.Subtitle, error) {
	video, ok := v.(polochon.Video)
	if !ok {
		return nil, ErrInvalidArgument
	}

	sub := polochon.NewSubtitleFromVideo(video, lang)
	sub.Data = []byte("subtitle in " + string(lang))

	return sub, nil
}
