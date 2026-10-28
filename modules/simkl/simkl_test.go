package simkl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
)

func testModule(t *testing.T, p Params, h http.HandlerFunc) *Simkl {
	t.Helper()
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	s := &Simkl{}
	raw := []byte(fmt.Sprintf("client_id: %q\nrefresh_token: %q\nclient_secret: %q\n", p.ClientID, p.RefreshToken, p.ClientSecret))
	if err := s.Init(raw, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatal(err)
	}
	s.client.http = server.Client()
	s.client.api = server.URL
	s.client.cdn = server.URL
	s.client.pace = 0
	s.client.sleep = func(ctx context.Context, _ time.Duration) error { return ctx.Err() }
	return s
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
func catalogRecord(id int, imdb, kind string) record {
	return record{Title: "Example", Year: "2020", Type: kind, IDs: ids{Simkl: number(fmt.Sprint(id)), IMDb: imdb}}
}
func tokenJSON(w http.ResponseWriter, token string) {
	writeJSON(w, map[string]any{"access_token": token, "refresh_token": "refresh-secret", "expires_in": 604800, "token_type": "Bearer", "scope": "media:read"})
}

func TestPublicCatalogAndDetailRoundTrip(t *testing.T) {
	var details, episodes atomic.Int32
	s := testModule(t, Params{ClientID: "client", RefreshToken: "refresh-secret"}, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("public request has account credentials")
		}
		if r.Header.Get("User-Agent") == "" || r.URL.Query().Get("app-name") != "polochon" || r.URL.Query().Get("app-version") == "" || r.URL.Query().Get("client_id") != "client" {
			t.Error("missing app identification")
		}
		switch r.URL.Path {
		case "/discover/trending/movies/today_100.json":
			a := catalogRecord(1, "tt1", "movie")
			a.Runtime = "1h 59m"
			a.ReleaseDate = "08/26/2026"
			a.Year = ""
			a.Poster = "74/poster"
			a.Fanart = "45/background"
			a.Ratings = map[string]rating{"simkl": {Rating: "7.5", Votes: "12"}}
			b := catalogRecord(2, "", "movie")
			wrong := catalogRecord(3, "tt3", "anime")
			writeJSON(w, []record{a, b, wrong, catalogRecord(4, "tt4", "movie")})
		case "/movies/1":
			details.Add(1)
			a := catalogRecord(1, "tt1", "movie")
			a.Title = "Detailed"
			writeJSON(w, a)
		case "/discover/trending/tv/today_100.json":
			writeJSON(w, []record{catalogRecord(10, "tt10", "show")})
		case "/tv/10":
			show := catalogRecord(10, "tt10", "show")
			show.IDs.TVDB = "42"
			writeJSON(w, show)
		case "/tv/episodes/10":
			episodes.Add(1)
			writeJSON(w, []episodeRecord{{Title: "Pilot", Season: "1", Episode: "1", IDs: ids{SimklID: "100"}, Date: "2020-01-01T21:00:00-05:00"}})
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			w.WriteHeader(404)
		}
	})
	ctx := context.Background()
	list, err := s.GetMovieList(ctx, "simkl-trending-today")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ImdbID != "tt1" || list[1].ImdbID != "tt4" || list[0].Runtime != 119 || list[0].Year != 2026 || list[0].Rating != 7.5 || list[0].Votes != 12 {
		t.Fatalf("unexpected conversion: %+v", list)
	}
	if !strings.HasSuffix(list[0].ThumbURL, "_w.webp") || !strings.HasSuffix(list[0].FanartURL, "_d.jpg") {
		t.Fatal("incorrect artwork format")
	}
	movie := &polochon.Movie{ImdbID: "tt1", Plot: "preserve", ThumbURL: "https://existing/poster.jpg"}
	for range 2 {
		if err = s.GetDetails(ctx, movie); err != nil {
			t.Fatal(err)
		}
	}
	if movie.Title != "Detailed" || movie.Plot != "preserve" || movie.ThumbURL != "https://existing/poster.jpg" || details.Load() != 1 {
		t.Fatalf("detail merge/cache failed: %+v", movie)
	}
	shows, err := s.GetShowList(ctx, "simkl-trending-today")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.GetDetails(ctx, shows[0]); err != nil {
		t.Fatal(err)
	}
	ep := &polochon.ShowEpisode{ShowImdbID: "tt10", Season: 1, Episode: 1, Plot: "preserved episode plot"}
	if err = s.GetDetails(ctx, ep); err != nil {
		t.Fatal(err)
	}
	if ep.Title != "Pilot" || ep.Show == nil || ep.ShowImdbID != "tt10" || ep.ShowTvdbID != 42 || ep.EpisodeImdbID != "" || ep.Plot != "preserved episode plot" || episodes.Load() != 1 {
		t.Fatalf("episode conversion or shared cache failed: %+v", ep)
	}
	if err = s.Init([]byte("invalid: ["), nil); err != nil {
		t.Fatal("repeated role initialization must be idempotent", err)
	}
	cold := newClient(Params{})
	s.client = cold
	if err = s.GetDetails(ctx, &polochon.Movie{ImdbID: "tt1"}); !errors.Is(err, ErrConnectionRequired) {
		t.Fatalf("cold identity lookup: %v", err)
	}
}

func TestCatalogOptionsAndIdentityConflicts(t *testing.T) {
	var requests []string
	s := testModule(t, Params{}, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		switch r.URL.Path {
		case "/discover/trending/movies/genre/index.json":
			writeJSON(w, []genre{{Name: "Not the slug", Slug: "science-fiction"}, {Slug: "../../bad"}})
		case "/discover/trending/movies/genre/science-fiction_month_500.json", "/discover/dvd/releases_500.json", "/discover/trending/movies/week_500.json":
			writeJSON(w, []record{catalogRecord(1, "tt1", "movie"), catalogRecord(2, "tt1", "movie"), catalogRecord(3, "tt3", "movie")})
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			w.WriteHeader(404)
		}
	})
	options := s.AvailableMovieOptions()
	found := false
	for _, o := range options {
		if o == "simkl-genre:science-fiction" {
			found = true
		}
		if strings.Contains(o, "bad") {
			t.Fatal("invalid genre advertised")
		}
	}
	if !found {
		t.Fatal("index-derived genre missing")
	}
	for _, option := range []string{"simkl-genre:science-fiction", "simkl-dvd-500", "simkl-trending-week-500"} {
		list, err := s.GetMovieList(context.Background(), option)
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 1 || list[0].ImdbID != "tt3" {
			t.Fatal("conflicting canonical records escaped filtering")
		}
	}
	before := len(requests)
	for _, option := range []string{"simkl-trending-year", "simkl-genre:../../secret", "anime"} {
		if _, err := s.GetMovieList(context.Background(), option); !errors.Is(err, ErrUnsupported) {
			t.Fatalf("invalid option: %v", err)
		}
	}
	if len(requests) != before {
		t.Fatal("invalid options sent upstream")
	}
	if err := s.GetDetails(context.Background(), &polochon.Movie{ImdbID: "tt1"}); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("expected ambiguous mapping: %v", err)
	}
}

func TestSearchRequiresConnectionAndUsesAuthenticatedEndpoint(t *testing.T) {
	var calls atomic.Int32
	s := testModule(t, Params{}, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); t.Error("anonymous search issued HTTP") })
	if _, err := s.SearchMovie(context.Background(), "Example"); !errors.Is(err, ErrConnectionRequired) {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Fatal("search fell back to discovery")
	}
	s = testModule(t, Params{ClientID: "client", RefreshToken: "refresh-secret"}, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			tokenJSON(w, "access-secret")
			return
		}
		if r.URL.Path != "/search/movie" || r.URL.Query().Get("q") != "Example" || r.Header.Get("Authorization") != "Bearer access-secret" {
			t.Errorf("incorrect search: %s", r.URL.Path)
		}
		a := catalogRecord(1, "tt1", "movie")
		a.Type = ""
		a.EndpointType = "movies"
		writeJSON(w, []record{a})
	})
	results, err := s.SearchMovie(context.Background(), "Example")
	if err != nil || len(results) != 1 {
		t.Fatalf("%+v %v", results, err)
	}
}

func TestCalendarAdapterFiltersAtReadTime(t *testing.T) {
	var count atomic.Int32
	s := testModule(t, Params{}, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tv/10":
			a := catalogRecord(10, "tt10", "show")
			a.Airs.Timezone = "America/New_York"
			writeJSON(w, a)
		case "/tv/episodes/10":
			count.Add(1)
			no, yes := false, true
			writeJSON(w, []episodeRecord{
				{Season: "1", Episode: "1", Aired: &yes, Date: "2020-01-01T00:00:00Z"},
				{Season: "1", Episode: "2", Aired: &no, Date: "2020-01-01T00:00:00Z"},
				{Season: "1", Episode: "3", Aired: &yes},
				{Season: "1", Episode: "4", Date: "2026-03-08T08:00:00Z"},
				{Season: "1", Episode: "5", Date: "2026-03-08"},
				{Season: "1", Episode: "6", Date: "invalid"},
				{Type: "special", Title: "Unnumbered special", Date: "2020-01-01T00:00:00Z"},
			})
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			w.WriteHeader(404)
		}
	})
	now := time.Date(2026, 3, 8, 7, 30, 0, 0, time.UTC)
	s.client.now = func() time.Time { return now }
	s.client.remember("tv", catalogRecord(10, "tt10", "show"))
	show := &polochon.Show{ImdbID: "tt10"}
	calendar, err := s.GetShowCalendar(context.Background(), show)
	if err != nil {
		t.Fatal(err)
	}
	if len(calendar.Episodes) != 1 || calendar.Episodes[0].Episode != 1 || len(show.Episodes) != 6 {
		t.Fatalf("unsafe initial calendar: %+v", calendar.Episodes)
	}
	now = now.Add(40 * time.Minute)
	calendar, err = s.GetShowCalendar(context.Background(), show)
	if err != nil {
		t.Fatal(err)
	}
	if count.Load() != 1 || len(calendar.Episodes) != 2 || calendar.Episodes[1].Episode != 4 {
		t.Fatal("cached future episode did not become available")
	}
	for _, tc := range []struct {
		date, zone, at string
		want           bool
	}{
		{"2026-03-08", "America/New_York", "2026-03-09T03:59:59Z", false},
		{"2026-03-08", "America/New_York", "2026-03-09T04:00:00Z", true},
		{"2026-11-01", "America/New_York", "2026-11-02T04:59:59Z", false},
		{"2026-11-01", "America/New_York", "2026-11-02T05:00:00Z", true},
		{"2020-01-01", "", "2026-01-01T00:00:00Z", false},
		{"2026-01-01T20:00:00-05:00", "", "2026-01-02T00:59:59Z", false},
	} {
		at, _ := time.Parse(time.RFC3339, tc.at)
		if _, got := availableDate(tc.date, tc.zone, at); got != tc.want {
			t.Errorf("availability %v = %v", tc, got)
		}
	}
}

func TestConcurrentRoleReadsDeduplicate(t *testing.T) {
	var calls atomic.Int32
	s := testModule(t, Params{}, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeJSON(w, []record{catalogRecord(1, "tt1", "movie")})
	})
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if _, err := s.GetMovieList(context.Background(), "simkl-trending-today"); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("got %d requests", calls.Load())
	}
}

func TestLivePublicMovieTV(t *testing.T) {
	if os.Getenv("SIMKL_LIVE") != "1" {
		t.Skip("set SIMKL_LIVE=1 for anonymous read-only smoke tests")
	}
	s := &Simkl{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	movies, err := s.GetMovieList(ctx, "simkl-trending-today")
	if err != nil || len(movies) == 0 {
		t.Fatalf("movies: %v", err)
	}
	if err = s.GetDetails(ctx, movies[0]); err != nil {
		t.Fatal(err)
	}
	shows, err := s.GetShowList(ctx, "simkl-trending-today")
	if err != nil || len(shows) == 0 {
		t.Fatalf("TV: %v", err)
	}
	if err = s.GetDetails(ctx, shows[0]); err != nil {
		t.Fatal(err)
	}
	if _, err = s.GetShowCalendar(ctx, shows[0]); err != nil {
		t.Fatal(err)
	}
	for _, option := range []string{"simkl-dvd", "simkl-genre:science-fiction"} {
		if _, err = s.GetMovieList(ctx, option); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = s.GetShowList(ctx, "simkl-genre:drama"); err != nil {
		t.Fatal(err)
	}
}
