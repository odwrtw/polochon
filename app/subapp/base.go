package subapp

import (
	"errors"
	"log/slog"
	"sync"
)

// ErrPanicRecovered is returned if the app paniced and recovered
var ErrPanicRecovered = errors.New("subapp: panic recovered")

// Base represents the base of a sub app
type Base struct {
	AppName   string
	AppStatus Status
	log       *slog.Logger

	Done chan struct{}
	Wg   sync.WaitGroup
}

// NewBase returns a new base app
func NewBase(name string, log *slog.Logger) *Base {
	return &Base{
		AppName:   name,
		AppStatus: Unknown,
		log:       log,
	}
}

// InitStart inits the app, so that it can be started
func (b *Base) InitStart() {
	b.log.Info("starting app", "app_name", b.AppName)
	b.Done = make(chan struct{})
	b.AppStatus = Started
}

// Name returns the name of the app
func (b *Base) Name() string {
	return b.AppName
}

// Status returns the status of the app
func (b *Base) Status() Status {
	return b.AppStatus
}

// Stop stops the app
func (b *Base) Stop() {
	b.log.Debug("stopping app asynchronously", "app", b.AppName)
	if b.AppStatus == Started {
		close(b.Done)
		b.AppStatus = Stopped
	}
}

// BlockingStop stops the app and waits for it to be done
func (b *Base) BlockingStop() {
	b.log.Debug("stopping app synchronously", "app", b.AppName)

	// Send a signal to stop the app
	b.Stop()

	// Wait for all the goroutines to be done
	b.Wg.Wait()

	b.log.Info("app stopped", "app_name", b.AppName)
}
