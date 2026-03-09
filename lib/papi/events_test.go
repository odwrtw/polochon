package papi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestEventsReceive(t *testing.T) {
	var mu sync.Mutex
	var flush func()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flusher", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher.Flush()

		mu.Lock()
		flush = func() {
			_, _ = io.WriteString(w, "data:\n\n")
			flusher.Flush()
		}
		mu.Unlock()

		<-r.Context().Done()
	}))

	c, err := New(ts.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		ts.Close()
	}()

	ch := c.Events(ctx)

	// Wait for flush to be set (client connected).
	deadline := time.After(2 * time.Second)
	for {
		mu.Lock()
		f := flush
		mu.Unlock()
		if f != nil {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for client to connect")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	mu.Lock()
	flush()
	mu.Unlock()

	select {
	case _, ok := <-ch:
		if !ok {
			t.Fatal("channel closed unexpectedly")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive event")
	}
}

func TestEventsContextCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		<-r.Context().Done()
	}))
	defer ts.Close()

	c, err := New(ts.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	ch := c.Events(ctx)
	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected channel to be closed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("channel not closed after context cancel")
	}
}

func TestEventsReconnect(t *testing.T) {
	var mu sync.Mutex
	var connCount int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		connCount++
		n := connCount
		mu.Unlock()

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flusher", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")

		if n == 1 {
			// First connection: send one event then disconnect.
			_, _ = io.WriteString(w, "data:\n\n")
			flusher.Flush()
			return
		}

		// Second connection: send an event and keep alive.
		_, _ = io.WriteString(w, "data:\n\n")
		flusher.Flush()
		<-r.Context().Done()
	}))

	c, err := New(ts.URL)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		ts.Close()
	}()

	ch := c.Events(ctx)

	// Should receive events from first connection.
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive first event")
	}

	// Should reconnect and receive event from second connection.
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("did not receive event after reconnect")
	}

	mu.Lock()
	if connCount < 2 {
		t.Fatalf("expected at least 2 connections, got %d", connCount)
	}
	mu.Unlock()
}
