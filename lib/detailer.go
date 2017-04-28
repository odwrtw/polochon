package polochon

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/errors"
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

// Detailable represents a ressource which can be detailed
type Detailable interface {
	GetDetailers() []Detailer
}

// GetDetails helps getting infos for a Detailable object
// If there is an error, it will be of type *errors.Collector
func GetDetails(v Detailable, log *logrus.Entry) error {
	c := errors.NewCollector()

	detailers := v.GetDetailers()
	if len(detailers) == 0 {
		c.Push(errors.Wrap("No detailer available").Fatal())
		return c
	}

	var done bool
	for _, d := range detailers {
		detailerLog := log.WithField("detailer", d.Name())
		err := d.GetDetails(v, detailerLog)
		if err == nil {
			done = true
			break
		}
		c.Push(errors.Wrap(err).Ctx("Detailer", d.Name()))
	}
	if !done {
		c.Push(errors.Wrap("All detailers failed").Fatal())
	}

	if c.HasErrors() {
		return c
	}

	return nil
}
