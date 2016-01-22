package subapp

import "github.com/Sirupsen/logrus"

// Status represents the status of the app
type Status int

// Available statuses
const (
	Unknown Status = iota
	Started
	Stopped
)

// App represents an application launched by main application
type App interface {
	// Status returns the current sub app status
	Status() Status
	// Name returns the name of the sub app
	Name() string
	// Run starts the SubApp, it should be a synchronous call
	Run(log *logrus.Entry) error
	// Stop sends a signal to the SubApp to stop gracefully, this should be an
	// asynchronous call
	Stop(log *logrus.Entry)
	// BlockingStop is similar to Stop, except that it blocks until the sub app is stopped
	BlockingStop(log *logrus.Entry)
}
