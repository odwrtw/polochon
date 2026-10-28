package simkl

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
)

func testModule(t *testing.T, handler http.HandlerFunc) *Simkl {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		for key, want := range map[string]string{
			"client_id": "test-client", "app-name": "polochon", "app-version": "test",
		} {
			if got := r.URL.Query().Get(key); got != want {
				t.Errorf("%s = %q, want %q", key, got, want)
			}
		}
		if r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("User-Agent") != "polochon/test" {
			t.Errorf("unexpected headers: %v", r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	s := &Simkl{}
	if err := s.InitWithParams(&Params{ClientID: "test-client", AppVersion: "test"}); err != nil {
		t.Fatal(err)
	}
	s.client.baseURL, s.client.dataURL = server.URL, server.URL
	s.client.http = server.Client()
	s.client.interval = 0
	return s
}

func TestInit(t *testing.T) {
	for _, params := range []*Params{nil, {}, {ClientID: " \t"}} {
		if err := (&Simkl{}).InitWithParams(params); err != ErrMissingClientID {
			t.Errorf("InitWithParams(%+v) = %v", params, err)
		}
	}
	s := &Simkl{}
	if err := s.Init([]byte("client_id: example\napp_version: 1.2\n"), nil); err != nil {
		t.Fatal(err)
	}
	client := s.client
	if s.Name() != "simkl" || client.clientID != "example" || client.appVersion != "1.2" || client.http.Timeout == 0 {
		t.Fatalf("unexpected configuration: %+v", s)
	}
	if err := s.Init([]byte("invalid: ["), nil); err != nil || s.client != client {
		t.Fatal("repeated initialization changed the client")
	}
	if err := s.InitWithParams(nil); err != nil || s.client != client {
		t.Fatal("repeated parameter initialization changed the client")
	}
	if err := (&Simkl{}).Init([]byte("invalid: ["), nil); err == nil {
		t.Fatal("invalid YAML accepted")
	}
	if err := (&Simkl{}).InitWithParams(&Params{ClientID: "id", AppVersion: "1\r\nX: y"}); err != ErrInvalidArgument {
		t.Fatalf("invalid version: %v", err)
	}
	s = &Simkl{}
	if err := s.InitWithParams(&Params{ClientID: " id "}); err != nil || s.client.appVersion != "dev" || s.client.clientID != "id" {
		t.Fatalf("default version or trimmed ID: %+v, %v", s, err)
	}
}

func TestStatus(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   string
		status int
		want   polochon.ModuleStatus
	}{
		{"ok", `{"title":"The Matrix","type":"movie","ids":{"simkl":53992}}`, 200, polochon.StatusOK},
		{"empty", `[]`, 200, polochon.StatusFail},
		{"null", `null`, 200, polochon.StatusFail},
		{"invalid metadata", `{}`, 200, polochon.StatusFail},
		{"malformed metadata", `{`, 200, polochon.StatusFail},
		{"invalid key", `{"error":"client_id_failed"}`, 412, polochon.StatusFail},
		{"server error", `failure`, 500, polochon.StatusFail},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := testModule(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/movies/53992" || len(r.URL.Query()) != 3 {
					t.Errorf("unexpected status request: %s", r.URL)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})
			status, err := s.Status()
			if status != tc.want || (err == nil) != (tc.want == polochon.StatusOK) {
				t.Errorf("Status() = %s, %v", status, err)
			}
		})
	}
	if status, err := (&Simkl{}).Status(); status != polochon.StatusFail || err != ErrMissingClientID {
		t.Errorf("unconfigured Status() = %s, %v", status, err)
	}
}

func TestClientErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   string
		status int
	}{
		{"bad json", `{`, 200},
		{"wrong shape", `{}`, 200},
		{"invalid key", `{"error":"client_id_failed"}`, 412},
		{"rate limit", `{"error":"rate_limit"}`, 429},
		{"server error", `failure`, 500},
		{"CDN not found", `Not Found`, 404},
		{"unexpected redirect", ``, 301},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := 0
			s := testModule(t, func(w http.ResponseWriter, _ *http.Request) {
				requests++
				w.Header().Set("Retry-After", "42")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})
			_, err := s.GetMovieList(context.Background(), "trending")
			if err == nil || requests != 1 {
				t.Fatalf("error = %v, requests = %d", err, requests)
			}
			if tc.status >= 400 {
				var apiErr *apiError
				if !errors.As(err, &apiErr) || apiErr.statusCode != tc.status || apiErr.retryAfter != "42" || apiErr.Error() == "" {
					t.Errorf("lost HTTP error details: %v", err)
				}
			}
		})
	}
}

func TestClientPacing(t *testing.T) {
	var mu sync.Mutex
	var times []time.Time
	s := testModule(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/redirect" {
			t.Errorf("unexpected pacing request: %s", r.URL)
		}
		mu.Lock()
		times = append(times, time.Now())
		mu.Unlock()
		w.Header().Set("Location", "https://simkl.com/movies/42/example")
		w.WriteHeader(http.StatusMovedPermanently)
	})
	s.client.interval = 20 * time.Millisecond
	var wg sync.WaitGroup
	for range 3 {
		wg.Go(func() {
			if _, err := s.client.resolve(context.Background(), "movies", url.Values{"imdb": {"tt001"}}); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	mu.Lock()
	defer mu.Unlock()
	if len(times) != 3 {
		t.Fatalf("requests = %d, want 3", len(times))
	}
	for i := 1; i < len(times); i++ {
		if gap := times[i].Sub(times[i-1]); gap < s.client.interval {
			t.Errorf("dynamic request gap = %s, want at least %s", gap, s.client.interval)
		}
	}
}

func TestClientCancellation(t *testing.T) {
	s := testModule(t, func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.GetMovieList(ctx, "trending"); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled feed = %v", err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := s.client.resolve(ctx, "movies", url.Values{"imdb": {"tt001"}}); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("in-flight cancellation = %v", err)
	}
	// Cancellation while queued behind another dynamic call.
	s.client.gate <- struct{}{}
	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := s.client.resolve(ctx, "movies", url.Values{"imdb": {"tt001"}}); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("queued cancellation = %v", err)
	}
	<-s.client.gate
	// Cancellation during the inter-request pacing delay.
	s.client.next = time.Now().Add(time.Hour)
	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := s.client.resolve(ctx, "movies", url.Values{"imdb": {"tt001"}}); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("pacing cancellation = %v", err)
	}
}
