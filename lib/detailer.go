package polochon

import (
	"context"
	"errors"
	"log/slog"
)

// ErrGettingDetails is returned if polochon failed to get details of the video
var ErrGettingDetails = errors.New("polochon: failed to get details")

// Detailer is the interface to get details on a video or a show
type Detailer interface {
	Module
	GetDetails(ctx context.Context, i any) error
}

// Detailable represents a ressource which can be detailed
type Detailable interface {
	GetDetailers() []Detailer
}

// GetDetails helps getting infos for a Detailable object
// If there is an error, it will be of type *errors.Collector
func GetDetails(ctx context.Context, v Detailable, log *slog.Logger) error {
	detailers := v.GetDetailers()
	if len(detailers) == 0 {
		log.Warn("no detailer available")
		return ErrGettingDetails
	}

	for _, d := range detailers {
		if err := d.GetDetails(ctx, v); err == nil {
			return nil
		}
	}

	log.Info("all detailers failed")
	return ErrGettingDetails
}
