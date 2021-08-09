package mock

import (
	"fmt"
	"io/ioutil"
	"strings"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// GetSubtitle implements the Detailer interface
func (mock *Mock) GetSubtitle(v interface{}, lang polochon.Language, log *logrus.Entry) (*polochon.Subtitle, error) {
	r := strings.NewReader(fmt.Sprintf("subtitle in %s", lang))

	video, ok := v.(polochon.Video)
	if !ok {
		return nil, ErrInvalidArgument
	}

	sub := polochon.NewSubtitleFromVideo(video, lang)
	sub.ReadCloser = ioutil.NopCloser(r)

	return sub, nil
}
