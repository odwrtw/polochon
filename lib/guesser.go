package polochon

import (
	"log"

	"github.com/Sirupsen/logrus"
)

// Guesser is an interface which allows to get inforamtions to create a video
// from a file
type Guesser interface {
	Guess(conf VideoConfig, file File) (Video, error)
}

// RegisterGuesser helps register a new detailer
func RegisterGuesser(name string, f func(params map[string]string, log *logrus.Entry) (Guesser, error)) {
	if _, ok := registeredModules.Guessers[name]; ok {
		log.Panicf("modules: %q of type %q is already registered", name, TypeGuesser)
	}

	// Register the module
	registeredModules.Guessers[name] = f
}
