package polochon

import (
	"fmt"

	"github.com/Sirupsen/logrus"
)

// Detailer is the interface to get details on a video or a show
type Detailer interface {
	Module
	GetDetails(i interface{}, log *logrus.Entry) error
}

// RegisterDetailer helps register a new detailer
func RegisterDetailer(name string, f func(params []byte) (Detailer, error)) {
	if _, ok := registeredModules.Detailers[name]; ok {
		panic(fmt.Sprintf("modules: %q of type %q is already registered", name, TypeDetailer))
	}
	// Register the module
	registeredModules.Detailers[name] = f
}
