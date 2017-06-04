package polochon

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// Explorer is the interface explore new videos from different sources
type Explorer interface {
	Module
	AvailableMovieOptions() []string
	GetMovieList(option string, log *logrus.Entry) ([]*Movie, error)
	AvailableShowOptions() []string
	GetShowList(option string, log *logrus.Entry) ([]*Show, error)
}

// RegisterExplorer helps register a new Explorer
func RegisterExplorer(name string, f func(params []byte) (Explorer, error)) {
	if _, ok := registeredModules.Explorers[name]; ok {
		panic(fmt.Sprintf("modules: %q of type %q is already registered", name, TypeExplorer))
	}

	// Register the module
	registeredModules.Explorers[name] = f
}
