package tvdb

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
)

func newSearchTestModule(t *testing.T, search http.HandlerFunc) *TvDB {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/login":
			_, _ = w.Write([]byte(`{"status":"success","data":{"token":"token"}}`))
		case "/search":
			if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer token" {
				t.Errorf("unexpected search request: method=%s authorization=%q", r.Method, r.Header.Get("Authorization"))
			}
			search(w, r)
		default:
			t.Errorf("unexpected request: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	client := newAPIClient("api-key", "")
	client.baseURL = server.URL
	client.http = server.Client()
	return &TvDB{client: client, configured: true}
}

func TestSearchShow(t *testing.T) {
	fixture, err := os.ReadFile("testdata/search.json")
	if err != nil {
		t.Fatal(err)
	}
	requests := 0
	module := newSearchTestModule(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if got := r.URL.Query(); got.Get("query") != "East of Eden" || got.Get("type") != "series" || got.Has("year") {
			t.Errorf("unexpected search parameters: %v", got)
		}
		_, _ = w.Write(fixture)
	})

	shows, err := module.SearchShow(context.Background(), "East of Eden")
	if err != nil {
		t.Fatalf("SearchShow() error = %v", err)
	}
	firstAired := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	secondAired := time.Date(1981, time.February, 8, 0, 0, 0, 0, time.UTC)
	want := []*polochon.Show{
		{
			Title:      "East of Eden (2026)",
			TvdbID:     430034,
			ImdbID:     "tt21097264",
			Year:       2026,
			Plot:       "In the farmlands of the Salinas Valley in California, successive generations of two families, the Trasks and the Hamiltons, are destined to reenact the fateful story of Cain and Abel.",
			FirstAired: &firstAired,
			PosterURL:  "https://artworks.thetvdb.com/banners/v4/series/430034/posters/6aa5d2429b5f2.jpg",
		},
		{
			Title:      "East of Eden (1981)",
			TvdbID:     180961,
			ImdbID:     "tt0083409",
			Year:       1981,
			Plot:       "A powerful eight-hour adaptation of John Steinbeck's 1952 generational saga stars Bruce Boxleitner and Timothy Bottoms as battling brothers reminiscent of Cain and Abel, and Jane Seymour as the malevolent young woman who toys with their emotions.",
			FirstAired: &secondAired,
			PosterURL:  "https://artworks.thetvdb.com/banners/posters/180961-2.jpg",
		},
	}
	if !reflect.DeepEqual(shows, want) {
		t.Errorf("SearchShow() = %+v, want %+v", shows, want)
	}
	if requests != 1 {
		t.Errorf("got %d search requests, want 1", requests)
	}
}

func TestSearchShowResults(t *testing.T) {
	for _, tc := range []struct {
		name     string
		data     string
		want     int
		wantYear int
		wantDate string
	}{
		{name: "empty", data: `[]`},
		{name: "missing IMDb ID", data: `[{"tvdb_id":"1","remote_ids":[{"id":"123","sourceName":"TheMovieDB.com"}]}]`},
		{name: "invalid TVDB IDs", data: `[{"tvdb_id":"invalid","remote_ids":[{"id":"tt123"}]},{"tvdb_id":"0","remote_ids":[{"id":"tt123"}]}]`},
		{
			name: "skip unusable results and tolerate missing metadata",
			data: `[{"tvdb_id":"1"},{"tvdb_id":"2","name":"Show","remote_ids":[{"id":"123","sourceName":"TheMovieDB.com"},{"id":"tt123","sourceName":"IMDB"}]}]`,
			want: 1,
		},
		{
			name:     "year from first air date",
			data:     `[{"tvdb_id":"2","first_air_time":"2026-10-01","remote_ids":[{"id":"tt123"}]}]`,
			want:     1,
			wantYear: 2026,
			wantDate: "2026-10-01",
		},
		{
			name:     "invalid optional date",
			data:     `[{"tvdb_id":"2","year":"2026","first_air_time":"unknown","remote_ids":[{"id":"tt123"}]}]`,
			want:     1,
			wantYear: 2026,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			module := newSearchTestModule(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"status":"success","data":` + tc.data + `}`))
			})
			shows, err := module.SearchShow(context.Background(), "Show")
			if tc.want == 0 {
				if !errors.Is(err, ErrShowNotFound) || shows != nil {
					t.Fatalf("SearchShow() = %v, %v; want nil, ErrShowNotFound", shows, err)
				}
				return
			}
			if err != nil || len(shows) != tc.want {
				t.Fatalf("SearchShow() = %v, %v; want %d results", shows, err, tc.want)
			}
			if shows[0].TvdbID != 2 || shows[0].ImdbID != "tt123" {
				t.Errorf("unexpected IDs: %+v", shows[0])
			}
			if shows[0].Year != tc.wantYear {
				t.Errorf("got year %d, want %d", shows[0].Year, tc.wantYear)
			}
			gotDate := ""
			if shows[0].FirstAired != nil {
				gotDate = shows[0].FirstAired.Format("2006-01-02")
			}
			if gotDate != tc.wantDate {
				t.Errorf("got first aired date %q, want %q", gotDate, tc.wantDate)
			}
		})
	}
}

func TestSearchShowErrors(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			module := newSearchTestModule(t, func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "search failed", status)
			})
			shows, err := module.SearchShow(context.Background(), "Show")
			if shows != nil {
				t.Errorf("got results on error: %v", shows)
			}
			if status == http.StatusNotFound {
				if !errors.Is(err, ErrShowNotFound) {
					t.Errorf("got %v, want ErrShowNotFound", err)
				}
			} else if !isHTTPStatus(err, status) {
				t.Errorf("got %v, want HTTP %d", err, status)
			}
		})
	}
}

func TestSearchShowCancellation(t *testing.T) {
	module := newSearchTestModule(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected search request after cancellation")
	})
	module.client.token = "token"
	module.client.tokenTime = time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := module.SearchShow(ctx, "Show"); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
}

func TestSearchMovie(t *testing.T) {
	requests := 0
	module := newSearchTestModule(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if got := r.URL.Query(); got.Get("query") != "Movie & sequel" || got.Get("type") != "movie" || got.Has("year") {
			t.Errorf("unexpected search parameters: %v", got)
		}
		_, _ = w.Write([]byte(`{
			"status":"success",
			"data":[
				{
					"tvdb_id":"100",
					"name":"Movie",
					"year":"2026",
					"overview":"Movie plot.",
					"image_url":"https://artworks.example/movie.jpg",
					"remote_ids":[
						{"id":"123","sourceName":"TheMovieDB.com"},
						{"id":"tt1234567","sourceName":"IMDB"}
					]
				},
				{
					"tvdb_id":"200",
					"name":"Movie sequel",
					"first_air_time":"2027-10-01",
					"remote_ids":[{"id":"tt2345678","sourceName":"IMDB"}]
				}
			]
		}`))
	})
	movies, err := module.SearchMovie(context.Background(), "Movie & sequel")
	if err != nil {
		t.Fatalf("SearchMovie() error = %v", err)
	}
	if len(movies) != 2 {
		t.Fatalf("got %d movies, want 2", len(movies))
	}
	want := []polochon.Movie{
		{
			Title:    "Movie",
			ImdbID:   "tt1234567",
			TmdbID:   123,
			Year:     2026,
			Plot:     "Movie plot.",
			ThumbURL: "https://artworks.example/movie.jpg",
		},
		{Title: "Movie sequel", ImdbID: "tt2345678", Year: 2027},
	}
	for i, movie := range movies {
		if !reflect.DeepEqual(*movie, want[i]) {
			t.Errorf("movie %d = %+v, want %+v", i, *movie, want[i])
		}
	}
	if requests != 1 {
		t.Errorf("got %d search requests, want 1", requests)
	}
}

func TestSearchMovieResults(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
		want int
	}{
		{name: "empty", data: `[]`},
		{name: "missing IMDb ID", data: `[{"remote_ids":[{"id":"123","sourceName":"TheMovieDB.com"}]}]`},
		{
			name: "skip missing IMDb IDs and tolerate invalid optional metadata",
			data: `[
				{"name":"No IMDb ID"},
				{"name":"Movie","year":"unknown","first_air_time":"unknown","remote_ids":[{"id":"tt123","sourceName":"IMDB"},{"id":"invalid","sourceName":"TheMovieDB.com"}]}
			]`,
			want: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			module := newSearchTestModule(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"status":"success","data":` + tc.data + `}`))
			})
			movies, err := module.SearchMovie(context.Background(), "Movie")
			if tc.want == 0 {
				if movies != nil || !errors.Is(err, ErrMovieNotFound) {
					t.Fatalf("SearchMovie() = %v, %v; want nil, ErrMovieNotFound", movies, err)
				}
				return
			}
			if err != nil || len(movies) != tc.want {
				t.Fatalf("SearchMovie() = %v, %v; want %d results", movies, err, tc.want)
			}
			want := polochon.Movie{Title: "Movie", ImdbID: "tt123"}
			if !reflect.DeepEqual(*movies[0], want) {
				t.Errorf("got movie %+v, want %+v", *movies[0], want)
			}
		})
	}
}

func TestSearchMovieErrors(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			module := newSearchTestModule(t, func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "search failed", status)
			})
			movies, err := module.SearchMovie(context.Background(), "Movie")
			if movies != nil {
				t.Errorf("got results on error: %v", movies)
			}
			if status == http.StatusNotFound {
				if !errors.Is(err, ErrMovieNotFound) {
					t.Errorf("got %v, want ErrMovieNotFound", err)
				}
			} else if !isHTTPStatus(err, status) {
				t.Errorf("got %v, want HTTP %d", err, status)
			}
		})
	}
}

func TestSearchMovieCancellation(t *testing.T) {
	module := newSearchTestModule(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected search request after cancellation")
	})
	module.client.token = "token"
	module.client.tokenTime = time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := module.SearchMovie(ctx, "Movie"); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
}

func TestSearcherRegistration(t *testing.T) {
	module, err := polochon.GetModule("tvdb", polochon.TypeSearcher)
	if err != nil {
		t.Fatalf("GetModule() error = %v", err)
	}
	if _, ok := module.(polochon.Searcher); !ok {
		t.Errorf("registered module %T does not implement Searcher", module)
	}
}
