package mock

import (
	"fmt"
	"io/ioutil"
	"strings"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// GetSubtitle implements the Detailer interface
func (mock *Mock) GetSubtitle(v interface{}, lang polochon.Language, log *logrus.Entry) (polochon.Subtitle, error) {
	sub := strings.NewReader(fmt.Sprintf("subtitle in %s", lang))

	return ioutil.NopCloser(sub), nil
}
