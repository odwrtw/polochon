package mock

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
)

func init() {
	polochon.RegisterSubtitler(moduleName, NewSubtitler)
}

// NewSubtitler is an helper to avoid passing bytes
func NewSubtitler(p []byte) (polochon.Subtitler, error) {
	return &Mock{}, nil
}

// GetSubtitle implements the Detailer interface
func (mock *Mock) GetSubtitle(v interface{}, lang polochon.Language, log *logrus.Entry) (polochon.Subtitle, error) {
	sub := strings.NewReader(fmt.Sprintf("subtitle in %s", lang))

	return ioutil.NopCloser(sub), nil
}
