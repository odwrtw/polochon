package subapp

import (
	"sync"

	"github.com/sirupsen/logrus"
)

// Base represents the base of a sub app
type Base struct {
	AppName   string
	AppStatus Status

	Done chan struct{}
	Wg   sync.WaitGroup
}

// NewBase returns a new base app
func NewBase(name string) *Base {
	return &Base{
		AppName:   name,
		AppStatus: Unknown,
	}
}

// InitStart inits the app, so that it can be started
func (b *Base) InitStart(log *logrus.Entry) {
	log.WithField("app_name", b.AppName).Info("starting app")
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
func (b *Base) Stop(log *logrus.Entry) {
	log.WithField("app", b.AppName).Debug("stopping app asynchronously")
	if b.AppStatus == Started {
		close(b.Done)
		b.AppStatus = Stopped
	}
}

// BlockingStop stops the app a nd waits for it to be done
func (b *Base) BlockingStop(log *logrus.Entry) {
	log.WithField("app", b.AppName).Debug("stopping app synchronously")

	// Send a signal to stop the app
	b.Stop(log)

	// Wait for all the goroutines to be done
	b.Wg.Wait()

	log.WithField("app_name", b.AppName).Info("app stopped")
}
