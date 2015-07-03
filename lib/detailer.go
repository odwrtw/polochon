package polochon

import (
	"log"

	"github.com/Sirupsen/logrus"
)

// Detailer is the interface to get details on a video or a show
type Detailer interface {
	GetDetails(i interface{}) error
}

// RegisterDetailer helps register a new detailer
func RegisterDetailer(name string, f func(params map[string]interface{}, log *logrus.Entry) (Detailer, error)) {
	if _, ok := registeredModules.Detailers[name]; ok {
		log.Panicf("modules: %q of type %q is already registered", name, TypeDetailer)
	}

	// Register the module
	registeredModules.Detailers[name] = f
}
