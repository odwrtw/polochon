package server

import (
	"io"
	"net/http"
	"sync"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

const sseModuleName = "sse"

// Compile-time assertion.
var _ polochon.Notifier = (*sseHub)(nil)

type sseHub struct {
	mu      sync.Mutex
	clients map[chan struct{}]struct{}
}

func newSSEHub() *sseHub {
	return &sseHub{
		clients: make(map[chan struct{}]struct{}),
	}
}

// Module interface.
func (h *sseHub) Init(_ []byte) error                    { return nil }
func (h *sseHub) Name() string                           { return sseModuleName }
func (h *sseHub) Status() (polochon.ModuleStatus, error) { return polochon.StatusOK, nil }

// Notifier interface.
func (h *sseHub) Notify(_ any, _ *logrus.Entry) error {
	h.broadcast()
	return nil
}

func (h *sseHub) subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *sseHub) unsubscribe(ch chan struct{}) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
	close(ch)
}

func (h *sseHub) broadcast() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		// Non-blocking send to coalesce duplicate events.
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	ch := s.hub.subscribe()
	defer s.hub.unsubscribe(ch)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher.Flush()

	const keepalive = 30 * time.Second
	timer := time.NewTimer(keepalive)
	defer timer.Stop()

	for {
		select {
		case <-ch:
			_, _ = io.WriteString(w, "data:\n\n")
			flusher.Flush()
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(keepalive)
		case <-timer.C:
			_, _ = io.WriteString(w, ": keepalive\n\n")
			flusher.Flush()
			timer.Reset(keepalive)
		case <-r.Context().Done():
			return
		}
	}
}
