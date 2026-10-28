package simkl

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConditionalCacheAndBoundedStaleFallback(t *testing.T) {
	var calls atomic.Int32
	s := testModule(t, Params{}, func(w http.ResponseWriter, r *http.Request) {
		switch calls.Add(1) {
		case 1:
			w.Header().Set("ETag", "original")
			w.Header().Set("Cache-Control", "max-age=60")
			writeJSON(w, []int{1})
		case 2:
			if r.Header.Get("If-None-Match") != "original" {
				t.Error("missing validator")
			}
			w.Header().Set("Cache-Control", "max-age=60")
			w.WriteHeader(304)
		default:
			w.WriteHeader(503)
		}
	})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.client.now = func() time.Time { return now }
	get := func() error {
		var out []int
		return s.client.get(context.Background(), s.client.cdn+"/feed", false, time.Hour, true, &out)
	}
	if err := get(); err != nil {
		t.Fatal(err)
	}
	now = now.Add(30 * time.Second)
	if err := get(); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("fresh cache refetched")
	}
	now = now.Add(time.Minute)
	if err := get(); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	if err := get(); err != nil {
		t.Fatal("expected bounded stale response", err)
	}
	now = now.Add(25 * time.Hour)
	if err := get(); err == nil {
		t.Fatal("stale data served after limit")
	}
}

func TestResponseErrorsAndQuotaIsolation(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		status int
		code   string
		target error
	}{
		{429, "rate_limit", ErrQuota}, {200, "user_limit_exceeded", ErrQuota}, {400, "RATE_LIMIT", ErrQuota}, {403, "insufficient_scope", ErrScope}, {403, "pro_required", ErrPlan}, {401, "invalid_grant", ErrCredentials}, {401, "user_token_required", ErrConnectionRequired},
	} {
		err := responseError([]byte(fmt.Sprintf(`{"error":%q}`, tc.code)), make(http.Header), tc.status, now)
		if !errors.Is(err, tc.target) {
			t.Errorf("%+v: %v", tc, err)
		}
	}
	var posts, private, public atomic.Int32
	s := testModule(t, Params{ClientID: "client", RefreshToken: "refresh-secret"}, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			posts.Add(1)
			tokenJSON(w, "access-secret")
		case "/private":
			private.Add(1)
			w.WriteHeader(429)
			writeJSON(w, map[string]string{"error": "user_limit_exceeded"})
		case "/public":
			public.Add(1)
			if r.Header.Get("Authorization") != "" {
				t.Error("credential leak")
			}
			writeJSON(w, []int{1})
		}
	})
	s.client.now = func() time.Time { return now }
	var out []int
	for range 2 {
		err := s.client.get(context.Background(), s.client.api+"/private", true, 0, false, &out)
		if !errors.Is(err, ErrQuota) {
			t.Fatal(err)
		}
	}
	if err := s.client.get(context.Background(), s.client.cdn+"/public", false, time.Hour, false, &out); err != nil {
		t.Fatal(err)
	}
	if posts.Load() != 1 || private.Load() != 1 || public.Load() != 1 {
		t.Fatalf("quota did not isolate operations: %d %d %d", posts.Load(), private.Load(), public.Load())
	}
	reset := responseError([]byte(`{"error":"user_limit_exceeded"}`), make(http.Header), 429, now).(*APIError).RetryAt
	if reset.UTC().Format(time.RFC3339) != "2026-11-02T05:00:00Z" {
		t.Fatalf("incorrect DST quota reset: %v", reset)
	}
	// Absent quota headers never mean exhausted; JSON error bodies still do.
	if err := responseError([]byte(`[]`), make(http.Header), 200, now); err != nil {
		t.Fatal(err)
	}
}

func TestCancellationLimitsAndSecretSafeErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"large", func(w http.ResponseWriter, r *http.Request) { _, _ = fmt.Fprint(w, strings.Repeat("x", maxBody+1)) }},
		{"json", func(w http.ResponseWriter, r *http.Request) { _, _ = fmt.Fprint(w, "not-json") }},
		{"error", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]string{"error": "invalid_client", "message": "refresh-secret"})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := testModule(t, Params{}, tc.handler)
			var out []int
			err := s.client.get(context.Background(), s.client.api+"/data", false, 0, false, &out)
			if err == nil || strings.Contains(err.Error(), "refresh-secret") {
				t.Fatalf("unsafe/missing error: %v", err)
			}
		})
	}
	s := testModule(t, Params{}, func(w http.ResponseWriter, r *http.Request) { t.Error("canceled request reached server") })
	s.client.gate <- struct{}{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out []int
	if err := s.client.get(ctx, s.client.api+"/data", false, 0, false, &out); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	<-s.client.gate
	if err := waitContext(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	s.client.http.CloseIdleConnections()
}

func TestRedirectDoesNotForwardCredentials(t *testing.T) {
	var destinationCalls atomic.Int32
	dest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { destinationCalls.Add(1) }))
	defer dest.Close()
	s := testModule(t, Params{ClientID: "client", RefreshToken: "refresh-secret"}, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			tokenJSON(w, "access-secret")
			return
		}
		http.Redirect(w, r, dest.URL, http.StatusFound)
	})
	// Use the production redirect policy with the test server's transport.
	s.client.http = newHTTPClient()
	var out []int
	if err := s.client.get(context.Background(), s.client.api+"/private", true, 0, false, &out); err == nil {
		t.Fatal("redirect accepted as data")
	}
	if destinationCalls.Load() != 0 {
		t.Fatal("followed credential-bearing redirect")
	}
}

func TestConcurrentRefreshAndExpiredTokenRetry(t *testing.T) {
	var posts atomic.Int32
	s := testModule(t, Params{ClientID: "client", RefreshToken: "refresh-secret"}, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "refresh-secret" {
				t.Error("incorrect refresh grant")
			}
			tokenJSON(w, fmt.Sprintf("access-%d", posts.Add(1)))
			return
		}
		if r.Header.Get("Authorization") == "Bearer access-1" {
			w.WriteHeader(401)
			writeJSON(w, map[string]string{"error": "expired_token"})
			return
		}
		writeJSON(w, []int{1})
	})
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if _, err := s.client.accessToken(context.Background(), ""); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if posts.Load() != 1 {
		t.Fatalf("refreshes: %d", posts.Load())
	}
	var out []int
	if err := s.client.get(context.Background(), s.client.api+"/private", true, 0, false, &out); err != nil {
		t.Fatal(err)
	}
	if posts.Load() != 2 {
		t.Fatalf("expected one refresh after rejected token: %d", posts.Load())
	}
	for range 10 {
		wg.Go(func() {
			if _, err := s.client.accessToken(context.Background(), "access-1"); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if posts.Load() != 2 {
		t.Fatal("concurrent rejection caused refresh storm")
	}
}

func TestCacheBoundsAndNoStore(t *testing.T) {
	c := newClient(Params{})
	for i := range 200 {
		c.store(fmt.Sprint(i), []byte(`[]`), make(http.Header), time.Hour, false)
	}
	if len(c.cache) > 128 {
		t.Fatal("unbounded cache")
	}
	c.store("secret", []byte(`[]`), http.Header{"Cache-Control": {"no-store"}}, time.Hour, false)
	if _, ok := c.cache["secret"]; ok {
		t.Fatal("cached no-store response")
	}
	for _, tc := range []struct{ raw, want string }{
		{`"42"`, "42"}, {`42`, "42"}, {`null`, ""},
	} {
		var n number
		if err := n.UnmarshalJSON([]byte(tc.raw)); err != nil || string(n) != tc.want {
			t.Fatalf("%s: %q %v", tc.raw, n, err)
		}
	}
}

func TestSuccessfulLastQuotaRequestAndBurstRetry(t *testing.T) {
	for _, mode := range []string{"zero", "missing", "burst"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			s := testModule(t, Params{ClientID: "client", RefreshToken: "refresh-secret"}, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/oauth2/token" {
					tokenJSON(w, "access")
					return
				}
				n := calls.Add(1)
				if mode == "zero" {
					w.Header().Set("X-RateLimit-Remaining", "0")
				}
				if mode == "burst" && n == 1 {
					w.Header().Set("Retry-After", "86400")
					w.WriteHeader(429)
					writeJSON(w, map[string]string{"error": "rate_limit"})
					return
				}
				writeJSON(w, []int{1})
			})
			var waits []time.Duration
			s.client.sleep = func(ctx context.Context, d time.Duration) error {
				if d > 0 {
					waits = append(waits, d)
				}
				return ctx.Err()
			}
			var out []int
			if err := s.client.get(context.Background(), s.client.api+"/first", true, 0, false, &out); err != nil {
				t.Fatal("last allowed call must succeed", err)
			}
			err := s.client.get(context.Background(), s.client.api+"/second", true, 0, false, &out)
			if mode == "zero" {
				if !errors.Is(err, ErrQuota) || calls.Load() != 1 {
					t.Fatalf("quota header ignored: %v, %d", err, calls.Load())
				}
			} else if err != nil {
				t.Fatal(err)
			}
			for _, d := range waits {
				if d > time.Minute {
					t.Fatalf("burst limit used daily reset: %v", d)
				}
			}
		})
	}
}
