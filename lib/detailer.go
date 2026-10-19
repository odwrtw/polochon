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

// ContinuingDetailer contributes partial details. A successful call does not
// complete GetDetails when ContinueOnSuccess returns true. Existing detailers
// need not implement this interface.
type ContinuingDetailer interface {
	Detailer
	ContinueOnSuccess() bool
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
		err := d.GetDetails(ctx, v)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err == nil {
			if continuing, ok := d.(ContinuingDetailer); ok && continuing.ContinueOnSuccess() {
				continue
			}
			return nil
		}

		log.Warn("detailer failed", "detailer", d.Name(), "error", err)
	}

	log.Info("all detailers failed")
	return ErrGettingDetails
}
