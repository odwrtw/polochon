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

// PartialDetailer contributes partial details: a successful call does not
// complete GetDetails.
type PartialDetailer interface {
	Detailer
	PartialDetails()
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
		if err := ctx.Err(); err != nil {
			return err
		}
		_, partial := d.(PartialDetailer)
		err := d.GetDetails(ctx, v)
		if err := ctx.Err(); err != nil {
			return err
		}
		if err == nil {
			if partial {
				continue
			}
			return nil
		}

		log.Warn("detailer failed", "detailer", d.Name(), "error", err)
	}

	log.Info("all detailers failed")
	return ErrGettingDetails
}
