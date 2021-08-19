package polochon

import (
	"errors"

	"github.com/sirupsen/logrus"
)

// ErrGettingDetails is returned if polochon failed to get details of the video
var ErrGettingDetails = errors.New("polochon: failed to get details")

// Detailer is the interface to get details on a video or a show
type Detailer interface {
	Module
	GetDetails(i interface{}, log *logrus.Entry) error
}

// Detailable represents a ressource which can be detailed
type Detailable interface {
	GetDetailers() []Detailer
}

// GetDetails helps getting infos for a Detailable object
// If there is an error, it will be of type *errors.Collector
func GetDetails(v Detailable, log *logrus.Entry) error {
	detailers := v.GetDetailers()
	if len(detailers) == 0 {
		log.Warn("No detailer available")
		return ErrGettingDetails
	}

	var done bool
	for _, d := range detailers {
		detailerLog := log.WithField("detailer", d.Name())
		err := d.GetDetails(v, detailerLog)
		if err == nil {
			done = true
			break
		}

		log.Warn(err)
	}
	if !done {
		log.Info("All detailers failed")
		return ErrGettingDetails
	}

	return nil
}
