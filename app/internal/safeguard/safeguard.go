package safeguard

import (
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/errors"
)

const (
	// MaxEventCount represents the max number of events allowed during the
	// MaxEventDelay period of time
	MaxEventCount = 4
	// MaxEventDelay is the delay before resetting the event counter
	MaxEventDelay = 10 * time.Second
)

// Safeguard prevents the app from entering in a panic loop
type Safeguard struct {
	event chan struct{}
	done  chan struct{}
	count int
	wg    sync.WaitGroup
}

// New retuns a new safeguard
func New() *Safeguard {
	return &Safeguard{
		event: make(chan struct{}),
		done:  make(chan struct{}),
	}
}

// Event sends an event to the safeguard
func (s *Safeguard) Event() {
	// Send the event in a goroutine to prevent deadlocks
	go func() {
		s.event <- struct{}{}
	}()
}

// Run runs the safeguard
func (s *Safeguard) Run(log *logrus.Entry) error {
	s.wg.Add(1)
	defer s.wg.Done()

	log = log.WithField("module", "safeguard")
	log.Debug("safeguard started")

	for {
		select {
		case <-s.done:
			log.Debug("safeguard stopped")
			return nil
		case <-s.event:
			// Increase the event count
			s.count++

			if s.count >= MaxEventCount {
				ctx := errors.Context{
					"current_count": s.count,
					"max_count":     MaxEventCount,
				}

				return errors.New("got %d safeguard events in less than %s",
					s.count, MaxEventDelay).Fatal().AddContext(ctx)
			}
		case <-time.After(MaxEventDelay):
			// Reset the panic count is there was not panic during the
			// MaxPanicDelay
			s.count = 0
		}
	}
}

// BlockingStop stops the safeguard
func (s *Safeguard) BlockingStop(log *logrus.Entry) {
	close(s.done)
	s.wg.Wait()
}
