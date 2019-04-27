package mock

import (
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// SearchMovie implements the searcher interface
func (mock *Mock) SearchMovie(key string, log *logrus.Entry) ([]*polochon.Movie, error) {
	return nil, nil
}

// SearchShow implements the searcher interface
func (mock *Mock) SearchShow(key string, log *logrus.Entry) ([]*polochon.Show, error) {
	return nil, nil
}
