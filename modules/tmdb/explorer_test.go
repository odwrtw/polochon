package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	tmdb "github.com/cyruzin/golang-tmdb"
	polochon "github.com/odwrtw/polochon/lib"
)

func moviePopularJSON(t *testing.T) *tmdb.MoviePopular {
	t.Helper()
	var r tmdb.MoviePopular
	if err := json.Unmarshal([]byte(`{"results":[{"id":42,"title":"The Matrix","overview":"A plot","release_date":"1999-03-30","poster_path":"/p.jpg","backdrop_path":"/b.jpg","vote_average":7.6}]}`), &r); err != nil {
		t.Fatal(err)
	}
	return &r
}

func tvAiringTodayJSON(t *testing.T) *tmdb.TVAiringToday {
	t.Helper()
	var r tmdb.TVAiringToday
	if err := json.Unmarshal([]byte(`{"results":[{"id":7,"name":"Example","overview":"Show plot","first_air_date":"2020-01-02","poster_path":"/s.jpg","backdrop_path":"/sb.jpg","vote_average":8.5}]}`), &r); err != nil {
		t.Fatal(err)
	}
	return &r
}

func TestTmdbSearchMovieIsLightweight(t *testing.T) {
	oldSearch, oldInfo := tmdbSearchMovie, tmdbGetMovieInfo
	t.Cleanup(func() {
		tmdbSearchMovie = oldSearch
		tmdbGetMovieInfo = oldInfo
	})

	tmdbSearchMovie = func(_ *tmdb.Client, _ string, _ map[string]string) (*tmdb.SearchMovies, error) {
		return &tmdb.SearchMovies{
			SearchMoviesResults: &tmdb.SearchMoviesResults{
				Results: []tmdb.MovieResult{
					{ID: 42, Title: "The Matrix", Overview: "A plot", ReleaseDate: "1999-03-30", PosterPath: "/p.jpg", BackdropPath: "/b.jpg", VoteMetrics: tmdb.VoteMetrics{VoteAverage: 7.6}},
				},
			},
		}, nil
	}
	// Hydration must never happen for search results.
	tmdbGetMovieInfo = func(_ *tmdb.Client, _ int, _ map[string]string) (*tmdb.MovieDetails, error) {
		t.Fatal("detail wrapper must not be called during lightweight search")
		return nil, nil
	}

	res, err := mockTmdb.SearchMovie(context.Background(), "Matrix")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
	m := res[0]
	if m.TmdbID != 42 || m.Title != "The Matrix" || m.Plot != "A plot" || m.Year != 1999 || m.Rating == 0 {
		t.Fatalf("movie search fields not mapped: %+v", m)
	}
	if m.ThumbURL != TmDBimageBaseURL+"/p.jpg" || m.FanartURL != TmDBimageBaseURL+"/b.jpg" {
		t.Fatalf("movie search artwork not mapped: %+v", m)
	}
}

func TestTmdbSearchMovieEmpty(t *testing.T) {
	oldSearch := tmdbSearchMovie
	t.Cleanup(func() { tmdbSearchMovie = oldSearch })
	tmdbSearchMovie = func(_ *tmdb.Client, _ string, _ map[string]string) (*tmdb.SearchMovies, error) {
		return &tmdb.SearchMovies{SearchMoviesResults: &tmdb.SearchMoviesResults{}}, nil
	}
	if _, err := mockTmdb.SearchMovie(context.Background(), "nothing"); err != ErrNoMovieFound {
		t.Fatalf("got %v, want %v", err, ErrNoMovieFound)
	}
}

func TestTmdbSearchShowIsLightweight(t *testing.T) {
	oldSearch, oldInfo := tmdbSearchTVShow, tmdbGetTVInfo
	t.Cleanup(func() {
		tmdbSearchTVShow = oldSearch
		tmdbGetTVInfo = oldInfo
	})

	tmdbSearchTVShow = func(_ *tmdb.Client, _ string, _ map[string]string) (*tmdb.SearchTVShows, error) {
		return &tmdb.SearchTVShows{
			SearchTVShowsResults: &tmdb.SearchTVShowsResults{
				Results: []tmdb.TVShowResult{
					{ID: 7, Name: "Example", Overview: "Show plot", FirstAirDate: "2020-01-02", PosterPath: "/s.jpg", BackdropPath: "/sb.jpg", VoteMetrics: tmdb.VoteMetrics{VoteAverage: 8.5}},
				},
			},
		}, nil
	}
	// Hydration must never happen for search results.
	tmdbGetTVInfo = func(_ *tmdb.Client, _ int, _ map[string]string) (*tmdb.TVDetails, error) {
		t.Fatal("detail wrapper must not be called during lightweight search")
		return nil, nil
	}

	res, err := mockTmdb.SearchShow(context.Background(), "Example")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
	s := res[0]
	if s.TmdbID != 7 || s.Title != "Example" || s.Plot != "Show plot" || s.Year != 2020 || s.Rating == 0 {
		t.Fatalf("show search fields not mapped: %+v", s)
	}
	if s.PosterURL != TmDBimageBaseURL+"/s.jpg" || s.FanartURL != TmDBimageBaseURL+"/sb.jpg" || s.BannerURL != s.FanartURL {
		t.Fatalf("show search artwork not mapped: %+v", s)
	}
}

func TestTmdbSearchShowEmptyNoKey(t *testing.T) {
	if _, err := mockTmdb.SearchShow(context.Background(), ""); err != ErrNoShowTitle {
		t.Fatalf("got %v, want %v", err, ErrNoShowTitle)
	}
}

func TestTmdbSearchShowEmpty(t *testing.T) {
	oldSearch := tmdbSearchTVShow
	t.Cleanup(func() { tmdbSearchTVShow = oldSearch })
	tmdbSearchTVShow = func(_ *tmdb.Client, _ string, _ map[string]string) (*tmdb.SearchTVShows, error) {
		return &tmdb.SearchTVShows{SearchTVShowsResults: &tmdb.SearchTVShowsResults{}}, nil
	}
	if _, err := mockTmdb.SearchShow(context.Background(), "nothing"); err != ErrNoShowFound {
		t.Fatalf("got %v, want %v", err, ErrNoShowFound)
	}
}

func TestTmdbExplorerMovieOptions(t *testing.T) {
	got := mockTmdb.AvailableMovieOptions()
	want := []string{"popular", "top_rated", "upcoming", "now_playing"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("movie options = %v, want %v", got, want)
	}
}

func TestTmdbExplorerShowOptions(t *testing.T) {
	got := mockTmdb.AvailableShowOptions()
	want := []string{"popular", "top_rated", "airing_today", "on_the_air"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("show options = %v, want %v", got, want)
	}
}

func TestTmdbExplorerInvalidOptions(t *testing.T) {
	if _, err := mockTmdb.GetMovieList(context.Background(), "not_an_option"); err != ErrInvalidOption {
		t.Fatalf("movie: got %v, want %v", err, ErrInvalidOption)
	}
	if _, err := mockTmdb.GetShowList(context.Background(), "not_an_option"); err != ErrInvalidOption {
		t.Fatalf("show: got %v, want %v", err, ErrInvalidOption)
	}
}

func TestTmdbExplorerMovieListMapping(t *testing.T) {
	oldPopular, oldInfo, oldExternal := tmdbGetMoviePopular, tmdbGetMovieInfo, tmdbGetMovieExternalIDs
	t.Cleanup(func() {
		tmdbGetMoviePopular = oldPopular
		tmdbGetMovieInfo = oldInfo
		tmdbGetMovieExternalIDs = oldExternal
	})
	tmdbGetMoviePopular = func(_ *tmdb.Client, _ map[string]string) (*tmdb.MoviePopular, error) {
		return moviePopularJSON(t), nil
	}
	tmdbGetMovieInfo = func(_ *tmdb.Client, _ int, _ map[string]string) (*tmdb.MovieDetails, error) {
		t.Fatal("detail wrapper must not be called by the explorer")
		return nil, nil
	}
	tmdbGetMovieExternalIDs = func(_ *tmdb.Client, id int, _ map[string]string) (*tmdb.MovieExternalIDs, error) {
		if id != 42 {
			t.Fatalf("TMDB ID = %d, want 42", id)
		}
		return &tmdb.MovieExternalIDs{IMDbID: "tt0133093"}, nil
	}

	res, err := mockTmdb.GetMovieList(context.Background(), movieOptionPopular)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
	m := res[0]
	if m.TmdbID != 42 || m.ImdbID != "tt0133093" || m.Title != "The Matrix" || m.Plot != "A plot" || m.Year != 1999 || m.Rating == 0 {
		t.Fatalf("movie explorer fields not mapped: %+v", m)
	}
	if m.ThumbURL != TmDBimageBaseURL+"/p.jpg" || m.FanartURL != TmDBimageBaseURL+"/b.jpg" {
		t.Fatalf("movie explorer artwork not mapped: %+v", m)
	}
}

func TestTmdbExplorerShowListMapping(t *testing.T) {
	oldAiring, oldInfo, oldExternal := tmdbGetTVAiringToday, tmdbGetTVInfo, tmdbGetTVExternalIDs
	t.Cleanup(func() {
		tmdbGetTVAiringToday = oldAiring
		tmdbGetTVInfo = oldInfo
		tmdbGetTVExternalIDs = oldExternal
	})
	tmdbGetTVAiringToday = func(_ *tmdb.Client, _ map[string]string) (*tmdb.TVAiringToday, error) {
		return tvAiringTodayJSON(t), nil
	}
	tmdbGetTVInfo = func(_ *tmdb.Client, _ int, _ map[string]string) (*tmdb.TVDetails, error) {
		t.Fatal("detail wrapper must not be called by the explorer")
		return nil, nil
	}
	tmdbGetTVExternalIDs = func(_ *tmdb.Client, id int, _ map[string]string) (*tmdb.TVExternalIDs, error) {
		if id != 7 {
			t.Fatalf("TMDB ID = %d, want 7", id)
		}
		return &tmdb.TVExternalIDs{IMDbID: "tt-example"}, nil
	}

	res, err := mockTmdb.GetShowList(context.Background(), showOptionAiringToday)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}
	s := res[0]
	if s.TmdbID != 7 || s.ImdbID != "tt-example" || s.Title != "Example" || s.Plot != "Show plot" || s.Year != 2020 || s.Rating == 0 {
		t.Fatalf("show explorer fields not mapped: %+v", s)
	}
	if s.PosterURL != TmDBimageBaseURL+"/s.jpg" || s.FanartURL != TmDBimageBaseURL+"/sb.jpg" || s.BannerURL != s.FanartURL {
		t.Fatalf("show explorer artwork not mapped: %+v", s)
	}
}

func TestTmdbExplorerSkipsMovieWithoutIMDbID(t *testing.T) {
	oldPopular, oldExternal := tmdbGetMoviePopular, tmdbGetMovieExternalIDs
	t.Cleanup(func() {
		tmdbGetMoviePopular = oldPopular
		tmdbGetMovieExternalIDs = oldExternal
	})
	tmdbGetMoviePopular = func(_ *tmdb.Client, _ map[string]string) (*tmdb.MoviePopular, error) {
		return moviePopularJSON(t), nil
	}
	tmdbGetMovieExternalIDs = func(_ *tmdb.Client, _ int, _ map[string]string) (*tmdb.MovieExternalIDs, error) {
		return &tmdb.MovieExternalIDs{}, nil
	}

	res, err := mockTmdb.GetMovieList(context.Background(), movieOptionPopular)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 0 {
		t.Fatalf("expected no IMDb-less results, got %d", len(res))
	}
}

func TestTmdbExplorerEmptyList(t *testing.T) {
	oldPopular := tmdbGetMoviePopular
	t.Cleanup(func() { tmdbGetMoviePopular = oldPopular })
	tmdbGetMoviePopular = func(_ *tmdb.Client, _ map[string]string) (*tmdb.MoviePopular, error) {
		return &tmdb.MoviePopular{}, nil
	}
	res, err := mockTmdb.GetMovieList(context.Background(), movieOptionPopular)
	if err != nil {
		t.Fatalf("empty list should not error, got %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("expected empty result, got %d", len(res))
	}
}

func TestTmdbExplorerEmptyWrappedLists(t *testing.T) {
	oldMovieTopRated, oldMovieUpcoming := tmdbGetMovieTopRated, tmdbGetMovieUpcoming
	oldTVPopular, oldTVTopRated, oldTVOnTheAir := tmdbGetTVPopular, tmdbGetTVTopRated, tmdbGetTVOnTheAir
	t.Cleanup(func() {
		tmdbGetMovieTopRated = oldMovieTopRated
		tmdbGetMovieUpcoming = oldMovieUpcoming
		tmdbGetTVPopular = oldTVPopular
		tmdbGetTVTopRated = oldTVTopRated
		tmdbGetTVOnTheAir = oldTVOnTheAir
	})

	tmdbGetMovieTopRated = func(_ *tmdb.Client, _ map[string]string) (*tmdb.MovieTopRated, error) {
		return &tmdb.MovieTopRated{}, nil
	}
	tmdbGetMovieUpcoming = func(_ *tmdb.Client, _ map[string]string) (*tmdb.MovieUpcoming, error) {
		return &tmdb.MovieUpcoming{}, nil
	}
	tmdbGetTVPopular = func(_ *tmdb.Client, _ map[string]string) (*tmdb.TVPopular, error) {
		return &tmdb.TVPopular{}, nil
	}
	tmdbGetTVTopRated = func(_ *tmdb.Client, _ map[string]string) (*tmdb.TVTopRated, error) {
		return &tmdb.TVTopRated{}, nil
	}
	tmdbGetTVOnTheAir = func(_ *tmdb.Client, _ map[string]string) (*tmdb.TVOnTheAir, error) {
		return &tmdb.TVOnTheAir{}, nil
	}

	for _, option := range []string{movieOptionTopRated, movieOptionUpcoming} {
		results, err := mockTmdb.GetMovieList(context.Background(), option)
		if err != nil {
			t.Fatalf("%s: empty list should not error, got %v", option, err)
		}
		if len(results) != 0 {
			t.Fatalf("%s: expected empty result, got %d", option, len(results))
		}
	}
	for _, option := range []string{showOptionPopular, showOptionTopRated, showOptionOnTheAir} {
		results, err := mockTmdb.GetShowList(context.Background(), option)
		if err != nil {
			t.Fatalf("%s: empty list should not error, got %v", option, err)
		}
		if len(results) != 0 {
			t.Fatalf("%s: expected empty result, got %d", option, len(results))
		}
	}
}

func TestTmdbExplorerUpstreamError(t *testing.T) {
	oldPopular := tmdbGetMoviePopular
	t.Cleanup(func() { tmdbGetMoviePopular = oldPopular })
	sentinel := errors.New("boom")
	tmdbGetMoviePopular = func(_ *tmdb.Client, _ map[string]string) (*tmdb.MoviePopular, error) {
		return nil, sentinel
	}
	if _, err := mockTmdb.GetMovieList(context.Background(), movieOptionPopular); !errors.Is(err, sentinel) {
		t.Fatalf("got %v, want %v", err, sentinel)
	}
}

func TestTmdbFindExternalSourceForwarded(t *testing.T) {
	var mu sync.Mutex
	var movieSource, showSource string
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		source := r.URL.Query().Get("external_source")
		if r.URL.Path == "/find/tt0133093" {
			movieSource = source
		}
		if r.URL.Path == "/find/tt0000007" {
			showSource = source
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"movie_results":[],"tv_results":[],"tv_episode_results":[],"tv_season_results":[],"person_results":[]}`))
	}))
	defer srv.Close()

	client, err := tmdb.Init("fakekey")
	if err != nil {
		t.Fatal(err)
	}
	client.SetCustomBaseURL(srv.URL)

	if _, err := tmdbSearchByImdbID(client, "tt0133093", "imdb_id", map[string]string{}); err != nil {
		t.Fatal(err)
	}
	if _, err := tmdbSearchByExternalID(client, "tt0000007", "imdb_id", map[string]string{}); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if calls != 2 {
		t.Fatalf("expected 2 external-ID requests, got %d", calls)
	}
	if movieSource != "imdb_id" {
		t.Fatalf("movie external_source = %q, want imdb_id", movieSource)
	}
	if showSource != "imdb_id" {
		t.Fatalf("show external_source = %q, want imdb_id", showSource)
	}
}

var _ polochon.Explorer = (*TmDB)(nil)
