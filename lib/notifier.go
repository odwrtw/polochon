package polochon

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

// Notifier is an interface to notify when a video is added
type Notifier interface {
	Module
	Notify(interface{}, *logrus.Entry) error
}

// RegisterNotifier helps register a new notifier
func RegisterNotifier(name string, f func(params []byte) (Notifier, error)) {
	if _, ok := registeredModules.Notifiers[name]; ok {
		panic(fmt.Sprintf("modules: %q of type %q is already registered", name, TypeNotifier))
	}

	// Register the module
	registeredModules.Notifiers[name] = f
}
