package polochon

import (
	"log"

	"github.com/Sirupsen/logrus"
)

// Notifier is an interface to notify when a video is added
type Notifier interface {
	Module
	Notify(i interface{}) error
}

// RegisterNotifier helps register a new notifier
func RegisterNotifier(name string, f func(params map[string]interface{}, log *logrus.Entry) (Notifier, error)) {
	if _, ok := registeredModules.Notifiers[name]; ok {
		log.Panicf("modules: %q of type %q is already registered", name, TypeNotifier)
	}

	// Register the module
	registeredModules.Notifiers[name] = f
}
