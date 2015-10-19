package polochon

import (
	"fmt"

	"github.com/Sirupsen/logrus"
)

// Guesser is an interface which allows to get inforamtions to create a video
// from a file
type Guesser interface {
	Module
	Guess(conf VideoConfig, file File, log *logrus.Entry) (Video, error)
}

// RegisterGuesser helps register a new detailer
func RegisterGuesser(name string, f func(params map[string]interface{}) (Guesser, error)) {
	if _, ok := registeredModules.Guessers[name]; ok {
		panic(fmt.Sprintf("modules: %q of type %q is already registered", name, TypeDetailer))
	}

	// Register the module
	registeredModules.Guessers[name] = f
}
