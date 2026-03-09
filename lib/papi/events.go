package papi

import (
	"bufio"
	"context"
	"net/http"
	"time"
)

// Events returns a channel that receives on each library-changed SSE event.
// The channel is closed when the context is cancelled. On disconnect, the
// client reconnects with exponential backoff (1s to 30s cap).
func (c *Client) Events(ctx context.Context) <-chan struct{} {
	ch := make(chan struct{}, 1)
	go c.eventLoop(ctx, ch)
	return ch
}

func (c *Client) eventLoop(ctx context.Context, ch chan struct{}) {
	defer close(ch)

	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		_ = c.streamEvents(ctx, ch)
		if ctx.Err() != nil {
			return
		}

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func (c *Client) streamEvents(ctx context.Context, ch chan<- struct{}) error {
	url := c.endpoint + "/events"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "papi")
	if c.token != "" {
		req.Header.Set("X-Auth-Token", c.token)
	}
	if c.basicAuth != nil {
		req.SetBasicAuth(c.basicAuth.username, c.basicAuth.password)
	}

	// Use a client without timeout for the long-lived SSE connection.
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return ErrResourceNotFound
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if scanner.Text() == "data:" {
			// Non-blocking send to coalesce duplicate events.
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}

	return scanner.Err()
}
