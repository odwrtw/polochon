package simkl

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrending(t *testing.T) {
	for _, kind := range []string{"movies", "tv"} {
		for option, period := range map[string]string{"trending": "today", "trending-week": "week", "trending-month": "month"} {
			t.Run(kind+"/"+option, func(t *testing.T) {
				s := testModule(t, func(w http.ResponseWriter, r *http.Request) {
					if want := "/discover/trending/" + kind + "/" + period + "_100.json"; r.URL.Path != want {
						t.Errorf("path = %s, want %s", r.URL.Path, want)
					}
					_, _ = w.Write([]byte(`[
						{"title":"First","release_date":"03/15/2026","runtime":"2h 37m","ids":{"simkl_id":1,"imdb":"tt001","tmdb":"12","tvdb":"34"}},
						{"title":"Duplicate","ids":{"simkl_id":1,"imdb":"tt001"}},
						{"title":"Same IMDb","ids":{"simkl_id":2,"imdb":"tt001"}},
						{"title":"Second","release_date":"01/01/2025","ids":{"simkl_id":3,"imdb":"tt002"}},
						{"title":"No identity"}, null
					]`))
				})
				var title, imdb string
				var year, count int
				if kind == "movies" {
					items, err := s.GetMovieList(context.Background(), option)
					if err != nil {
						t.Fatal(err)
					}
					count = len(items)
					if count > 0 {
						title, imdb, year = items[0].Title, items[0].ImdbID, items[0].Year
						if items[0].TmdbID != 12 {
							t.Errorf("TMDB = %d", items[0].TmdbID)
						}
					}
				} else {
					items, err := s.GetShowList(context.Background(), option)
					if err != nil {
						t.Fatal(err)
					}
					count = len(items)
					if count > 0 {
						title, imdb, year = items[0].Title, items[0].ImdbID, items[0].Year
						if items[0].TvdbID != 34 {
							t.Errorf("TVDB = %d", items[0].TvdbID)
						}
					}
				}
				if count != 2 || title != "First" || imdb != "tt001" || year != 2026 {
					t.Errorf("got %d results, first=%q %q %d", count, title, imdb, year)
				}
			})
		}
	}
}

func TestExplorerEmptyAndInvalid(t *testing.T) {
	for _, body := range []string{`[]`, `null`} {
		s := testModule(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) })
		movies, err := s.GetMovieList(context.Background(), "trending")
		if err != nil || movies == nil || len(movies) != 0 {
			t.Errorf("empty movies = %v, %v", movies, err)
		}
		shows, err := s.GetShowList(context.Background(), "trending")
		if err != nil || shows == nil || len(shows) != 0 {
			t.Errorf("empty shows = %v, %v", shows, err)
		}
	}
	s := testModule(t, func(_ http.ResponseWriter, _ *http.Request) { t.Error("unexpected request") })
	if _, err := s.GetMovieList(context.Background(), "invalid"); err != ErrInvalidArgument {
		t.Errorf("invalid movie option = %v", err)
	}
	if _, err := s.GetShowList(context.Background(), "invalid"); err != ErrInvalidArgument {
		t.Errorf("invalid show option = %v", err)
	}
	if _, err := (&Simkl{}).GetMovieList(context.Background(), "trending"); err != ErrMissingClientID {
		t.Errorf("unconfigured explorer = %v", err)
	}
	s = testModule(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"title":"Test","release_date":"02/30/2026","ids":{"simkl_id":1,"imdb":"tt001"}}]`))
	})
	if _, err := s.GetMovieList(context.Background(), "trending"); err != ErrInvalidResponse {
		t.Errorf("invalid feed date = %v", err)
	}
}

func TestDiscoveryOptions(t *testing.T) {
	for _, kind := range []string{"movies", "tv"} {
		t.Run(kind, func(t *testing.T) {
			paths := map[string]string{
				"trending":       "/discover/trending/" + kind + "/today_100.json",
				"trending-week":  "/discover/trending/" + kind + "/week_100.json",
				"trending-month": "/discover/trending/" + kind + "/month_100.json",
			}
			genrePath := "/movies/genres/all/movies/all/all/"
			options := (&Simkl{}).AvailableMovieOptions()
			if kind == "movies" {
				paths["dvd"] = "/discover/dvd/releases_100.json"
			} else {
				options = (&Simkl{}).AvailableShowOptions()
				genrePath = "/tv/genres/all/all/all/all/all/"
				for option, path := range map[string]string{
					"best": "/tv/best/all", "best-year": "/tv/best/year", "best-month": "/tv/best/month",
					"best-voted": "/tv/best/voted", "best-watched": "/tv/best/watched",
					"premieres": "/tv/premieres/new", "upcoming": "/tv/premieres/soon",
					"airing": "/tv/airing", "airing-tomorrow": "/tv/airing",
				} {
					paths[option] = path
				}
			}
			for _, sort := range []string{"popular-this-week", "popular-this-month", "popular-all-time", "rank", "release-date", "voted", "watched"} {
				paths[sort] = genrePath + sort
			}
			if len(options) != len(paths) {
				t.Fatalf("got %d options, want %d", len(options), len(paths))
			}
			seen := map[string]bool{}
			for _, option := range options {
				t.Run(option, func(t *testing.T) {
					path, ok := paths[option]
					if !ok || seen[option] {
						t.Fatalf("unmapped or duplicate option %q", option)
					}
					seen[option] = true
					s := testModule(t, func(w http.ResponseWriter, r *http.Request) {
						if r.URL.Path != path {
							t.Errorf("path = %s, want %s", r.URL.Path, path)
						}
						q := r.URL.Query()
						if path == genrePath+option || option == "premieres" || option == "upcoming" {
							if q.Get("page") != "1" || q.Get("limit") != "40" {
								t.Errorf("pagination = %v", q)
							}
						}
						if option == "airing" && q.Get("date") != "today" || option == "airing-tomorrow" && q.Get("date") != "tomorrow" {
							t.Errorf("airing date = %s", q.Get("date"))
						}
						_, _ = w.Write([]byte(`[]`))
					})
					var err error
					if kind == "movies" {
						_, err = s.GetMovieList(context.Background(), option)
					} else {
						_, err = s.GetShowList(context.Background(), option)
					}
					if err != nil {
						t.Fatal(err)
					}
				})
			}
		})
	}
	for _, tc := range []struct{ kind, option string }{{"tv", "dvd"}, {"movies", "airing"}, {"movies", "best"}, {"movies", "upcoming"}} {
		if _, _, _, err := explorerRequest(tc.kind, tc.option); err != ErrInvalidArgument {
			t.Errorf("accepted option %q for %s", tc.option, tc.kind)
		}
	}
}

func TestExplorerEnrichment(t *testing.T) {
	apiRequests, dataRequests := 0, 0
	s := testModule(t, func(w http.ResponseWriter, r *http.Request) {
		apiRequests++
		if r.URL.Path != "/movies/42" {
			t.Errorf("unexpected API request: %s", r.URL)
		}
		_, _ = w.Write([]byte(`{"title":"Hydrated","year":2026,"type":"movie","ids":{"simkl":42,"imdb":"tt001"}}`))
	})
	data := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dataRequests++
		if r.URL.Path != "/discover/dvd/releases_100.json" || r.URL.Query().Get("client_id") != "test-client" || r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected CDN request: %s", r.URL)
		}
		_, _ = w.Write([]byte(`[{"title":"Sparse feed item","ids":{"simkl_id":42}}]`))
	}))
	t.Cleanup(data.Close)
	s.client.dataURL = data.URL
	items, err := s.GetMovieList(context.Background(), "dvd")
	if err != nil || len(items) != 1 || items[0].Title != "Hydrated" || apiRequests != 1 || dataRequests != 1 {
		t.Errorf("enrichment: items=%+v error=%v API=%d CDN=%d", items, err, apiRequests, dataRequests)
	}
}

func TestExplorerLimit(t *testing.T) {
	s := testModule(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "[")
		for i := range 100 {
			if i != 0 {
				_, _ = fmt.Fprint(w, ",")
			}
			_, _ = fmt.Fprintf(w, `{"title":"Title","ids":{"simkl_id":%d,"imdb":"tt%07d"}}`, i+1, i+1)
		}
		_, _ = fmt.Fprint(w, "]")
	})
	items, err := s.GetMovieList(context.Background(), "trending")
	if err != nil || len(items) != 40 {
		t.Errorf("bounded explorer got %d items, %v", len(items), err)
	}
}
