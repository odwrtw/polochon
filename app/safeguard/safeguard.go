package safeguard

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
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

// New returns a new safeguard
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
func (s *Safeguard) Run(log *slog.Logger) error {
	s.wg.Add(1)
	defer s.wg.Done()

	log = log.With("module", "safeguard")
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
				return fmt.Errorf(
					"got %d safeguard events in less than %s",
					s.count, MaxEventDelay)
			}
		case <-time.After(MaxEventDelay):
			// Reset the panic count if there was no panic during the
			// MaxPanicDelay
			s.count = 0
		}
	}
}

// BlockingStop stops the safeguard
func (s *Safeguard) BlockingStop() {
	close(s.done)
	s.wg.Wait()
}
