package server

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSSEHubSubscribeBroadcast(t *testing.T) {
	hub := newSSEHub()

	ch1 := hub.subscribe()
	ch2 := hub.subscribe()

	hub.broadcast()

	select {
	case <-ch1:
	case <-time.After(time.Second):
		t.Fatal("ch1 did not receive broadcast")
	}
	select {
	case <-ch2:
	case <-time.After(time.Second):
		t.Fatal("ch2 did not receive broadcast")
	}

	hub.unsubscribe(ch1)
	hub.broadcast()

	// ch1 should be closed
	select {
	case _, ok := <-ch1:
		if ok {
			t.Fatal("ch1 should be closed")
		}
	default:
		t.Fatal("ch1 should be closed and readable")
	}

	// ch2 should still receive
	select {
	case <-ch2:
	case <-time.After(time.Second):
		t.Fatal("ch2 did not receive broadcast after ch1 unsubscribed")
	}

	hub.unsubscribe(ch2)
}

func TestSSEHubCoalescing(t *testing.T) {
	hub := newSSEHub()
	ch := hub.subscribe()
	defer hub.unsubscribe(ch)

	// Multiple broadcasts before consuming should coalesce into one.
	hub.broadcast()
	hub.broadcast()
	hub.broadcast()

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("did not receive coalesced broadcast")
	}

	// Channel should be empty now.
	select {
	case <-ch:
		t.Fatal("should not receive a second event (coalescing)")
	default:
	}
}

func TestSSEHandler(t *testing.T) {
	hub := newSSEHub()
	s := &Server{hub: hub}

	ts := httptest.NewServer(http.HandlerFunc(s.events))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("expected Content-Type text/event-stream, got %q", ct)
	}

	hub.broadcast()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if scanner.Text() == "data:" {
			return
		}
	}
	t.Fatal("did not receive SSE data line")
}
