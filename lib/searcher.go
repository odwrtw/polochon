package polochon

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// Searcher is the interface to search shows or movies from different sources
type Searcher interface {
	SearchMovie(key string, log *logrus.Entry) ([]*Movie, error)
	SearchShow(key string, log *logrus.Entry) ([]*Show, error)
}

// RegisterSearcher helps register a new searcher
func RegisterSearcher(name string, f func(params []byte) (Searcher, error)) {
	if _, ok := registeredModules.Searchers[name]; ok {
		panic(fmt.Sprintf("modules: %q of type %q is already registered", name, TypeSearcher))
	}

	// Register the module
	registeredModules.Searchers[name] = f
}
