package polochon

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// Guesser is an interface which allows to get inforamtions to create a video
// from a file
type Guesser interface {
	Module
	Guess(File, MovieConfig, ShowConfig, *logrus.Entry) (Video, error)
}

// RegisterGuesser helps register a new detailer
func RegisterGuesser(name string, f func(params []byte) (Guesser, error)) {
	if _, ok := registeredModules.Guessers[name]; ok {
		panic(fmt.Sprintf("modules: %q of type %q is already registered", name, TypeGuesser))
	}

	// Register the module
	registeredModules.Guessers[name] = f
}
