package polochon

import (
	"github.com/sirupsen/logrus"
)

// Searcher is the interface to search shows or movies from different sources
type Searcher interface {
	Module
	SearchMovie(key string, log *logrus.Entry) ([]*Movie, error)
	SearchShow(key string, log *logrus.Entry) ([]*Show, error)
}
