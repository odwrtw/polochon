package simkl

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
)

func TestSearch(t *testing.T) {
	for _, kind := range []string{"movies", "tv"} {
		t.Run(kind, func(t *testing.T) {
			searchType, detailType := "movie", "movie"
			if kind == "tv" {
				searchType, detailType = "tv", "show"
			}
			requests := 0
			s := testModule(t, func(w http.ResponseWriter, r *http.Request) {
				requests++
				switch r.URL.Path {
				case "/search/" + searchType:
					q := r.URL.Query()
					if q.Get("q") != "A & B + 日本語" || q.Get("page") != "1" || q.Get("limit") != "20" {
						t.Errorf("unexpected query: %v", q)
					}
					_, _ = fmt.Fprintf(w, `[
						{"title":"First","endpoint_type":%q,"ids":{"simkl_id":1}},
						{"title":"Duplicate","endpoint_type":%q,"ids":{"simkl_id":1}},
						{"title":"No IMDb","endpoint_type":%q,"ids":{"simkl_id":2}},
						{"title":"Missing","endpoint_type":%q,"ids":{"simkl_id":3}},
						{"title":"Second","endpoint_type":%q,"ids":{"simkl_id":4}}
					]`, kind, kind, kind, kind, kind)
				case "/" + kind + "/1", "/" + kind + "/4":
					id, title, imdb := 1, "First", "tt001"
					if r.URL.Path == "/"+kind+"/4" {
						id, title, imdb = 4, "Second", "tt004"
					}
					_, _ = fmt.Fprintf(w, `{"title":%q,"year":2026,"type":%q,"ids":{"simkl":%d,"imdb":%q,"tmdb":"12","tvdb":"34"}}`, title, detailType, id, imdb)
				case "/" + kind + "/2":
					_, _ = fmt.Fprintf(w, `{"title":"No IMDb","type":%q,"ids":{"simkl":2}}`, detailType)
				case "/" + kind + "/3":
					_, _ = w.Write([]byte(`[]`))
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
					http.NotFound(w, r)
				}
			})
			if kind == "movies" {
				items, err := s.SearchMovie(context.Background(), "A & B + 日本語")
				if err != nil {
					t.Fatal(err)
				}
				if len(items) != 2 || items[0].Title != "First" || items[0].ImdbID != "tt001" || items[1].Title != "Second" || items[0].TmdbID != 12 {
					t.Errorf("unexpected movies: %+v", items)
				}
			} else {
				items, err := s.SearchShow(context.Background(), "A & B + 日本語")
				if err != nil {
					t.Fatal(err)
				}
				if len(items) != 2 || items[0].Title != "First" || items[0].ImdbID != "tt001" || items[1].Title != "Second" || items[0].TvdbID != 34 {
					t.Errorf("unexpected shows: %+v", items)
				}
			}
			if requests != 5 {
				t.Errorf("requests = %d, want search + four unique details", requests)
			}
		})
	}
}

func TestSearchFailures(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want error
	}{
		{"empty", `[]`, ErrNotFound},
		{"null", `null`, ErrNotFound},
		{"wrong catalog", `[{"title":"Anime","endpoint_type":"anime","ids":{"simkl_id":1}}]`, ErrInvalidResponse},
		{"no identity", `[{"title":"Missing","endpoint_type":"movies"}]`, ErrNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := testModule(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(tc.body)) })
			if _, err := s.SearchMovie(context.Background(), "test"); err != tc.want {
				t.Errorf("search = %v, want %v", err, tc.want)
			}
		})
	}
	s := testModule(t, func(_ http.ResponseWriter, _ *http.Request) { t.Error("unexpected request") })
	if _, err := s.SearchMovie(context.Background(), " \n"); err != ErrInvalidArgument {
		t.Errorf("empty movie query = %v", err)
	}
	if _, err := s.SearchShow(context.Background(), ""); err != ErrInvalidArgument {
		t.Errorf("empty show query = %v", err)
	}
	if _, err := (&Simkl{}).SearchShow(context.Background(), "test"); err != ErrMissingClientID {
		t.Errorf("unconfigured search = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.SearchMovie(ctx, "test"); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled search = %v", err)
	}
}

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

func TestSearchEnrichmentError(t *testing.T) {
	requests := 0
	s := testModule(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path == "/search/movie" {
			_, _ = w.Write([]byte(`[
				{"title":"Already usable","endpoint_type":"movies","ids":{"simkl_id":1,"imdb":"tt001"}},
				{"title":"Needs hydration","endpoint_type":"movies","ids":{"simkl_id":2}}
			]`))
			return
		}
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate_limit"}`))
	})
	items, err := s.SearchMovie(context.Background(), "test")
	var apiErr *apiError
	if items != nil || !errors.As(err, &apiErr) || apiErr.statusCode != 429 || requests != 2 {
		t.Errorf("enrichment failure = %+v, %v, requests=%d", items, err, requests)
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
