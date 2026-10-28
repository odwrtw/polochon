package simkl

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func TestResolve(t *testing.T) {
	for _, tc := range []struct {
		name     string
		kind     string
		location string
		status   int
		wantID   int
	}{
		{"movie", "movies", "https://simkl.com/movies/42/example", 301, 42},
		{"show", "tv", "//simkl.com/tv/42/example", 301, 42},
		{"unknown", "movies", "//simkl.com", 301, 0},
		{"anime", "tv", "https://simkl.com/anime/42/example", 301, 0},
		{"wrong catalog", "movies", "https://simkl.com/tv/42/example", 301, 0},
		{"wrong host", "movies", "https://example.org/movies/42", 301, 0},
		{"credentials", "movies", "https://user@simkl.com/movies/42", 301, 0},
		{"wrong scheme", "movies", "http://simkl.com/movies/42", 301, 0},
		{"invalid ID", "movies", "https://simkl.com/movies/nope", 301, 0},
		{"zero ID", "movies", "https://simkl.com/movies/0", 301, 0},
		{"missing location", "movies", "", 301, 0},
		{"bad location", "movies", "%", 301, 0},
		{"not a redirect", "movies", "", 200, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := testModule(t, func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				if r.URL.Path != "/redirect" || q.Get("to") != "simkl" || q.Get("imdb") != "tt001" {
					t.Errorf("unexpected resolver request: %s", r.URL)
				}
				if tc.location != "" {
					w.Header().Set("Location", tc.location)
				}
				w.WriteHeader(tc.status)
			})
			// Any redirect follow would fail the test without making external requests.
			s.client.http.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
				t.Error("followed redirect")
				return http.ErrUseLastResponse
			}
			id, err := s.client.resolve(context.Background(), tc.kind, url.Values{"imdb": {"tt001"}})
			if id != tc.wantID || (err == nil) != (tc.wantID > 0) {
				t.Errorf("resolve = %d, %v", id, err)
			}
		})
	}
}

func TestDetailResponseValidation(t *testing.T) {
	for _, body := range []string{`[]`, `[ ]`, `null`, `[{}]`, `{}`, `{`, `{"title":"Wrong ID","type":"movie","ids":{"simkl":2}}`, `{"title":"Anime","type":"anime","ids":{"simkl":1}}`} {
		t.Run(body, func(t *testing.T) {
			s := testModule(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) })
			if _, err := s.client.detail(context.Background(), "movies", 1); err == nil {
				t.Fatal("invalid detail accepted")
			}
		})
	}
	s := testModule(t, func(_ http.ResponseWriter, _ *http.Request) { t.Error("unexpected request") })
	if _, err := s.client.detail(context.Background(), "movies", 0); err != ErrInvalidArgument {
		t.Errorf("invalid ID = %v", err)
	}
}
