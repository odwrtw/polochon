package tmdb

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"reflect"
	"testing"

	tmdb "github.com/cyruzin/golang-tmdb"
	polochon "github.com/odwrtw/polochon/lib"
)

// Fake TmDB pointer to run the tests
var mockTmdb = &TmDB{log: slog.Default()}

func fakeFindByID(movieID int64) *tmdb.FindByID {
	results := &tmdb.FindByID{}
	_ = json.Unmarshal([]byte(`{"movie_results":[{"id":1000}]}`), results)
	results.MovieResults[0].ID = movieID
	return results
}

func TestTmdbInvalidMovieArgument(t *testing.T) {
	m := polochon.NewShowEpisode(polochon.ShowConfig{})
	err := mockTmdb.GetDetails(context.Background(), m)
	if err != ErrInvalidArgument {
		log.Fatalf("Got %q, expected %q", err, ErrInvalidArgument)
	}
}

func TestTmdbMissingArgument(t *testing.T) {
	tm := &TmDB{}
	err := tm.Init(nil, slog.Default())
	if err != ErrMissingArgument {
		log.Fatalf("Got %q, expected %q", err, ErrMissingArgument)
	}
}

func TestTmdbInit(t *testing.T) {
	tm := &TmDB{}
	if err := tm.Init([]byte("apikey: test"), slog.Default()); err != nil {
		t.Fatalf("expected initialization to succeed, got %v", err)
	}
	if !tm.configured || tm.client == nil {
		t.Fatal("expected TMDB client to be initialized")
	}
}

func TestTmdbStatus(t *testing.T) {
	oldSearch := tmdbSearchByImdbID
	t.Cleanup(func() { tmdbSearchByImdbID = oldSearch })
	tmdbSearchByImdbID = func(_ *tmdb.Client, _, _ string, _ map[string]string) (*tmdb.FindByID, error) {
		return fakeFindByID(603), nil
	}

	status, err := (&TmDB{}).Status()
	if err != nil {
		t.Fatalf("expected status check to succeed, got %v", err)
	}
	if status != polochon.StatusOK {
		t.Fatalf("expected status OK, got %v", status)
	}
}

func TestTmdbSearchByTitleArguments(t *testing.T) {
	m := polochon.NewMovie(polochon.MovieConfig{})

	oldSearch := tmdbSearchMovie
	t.Cleanup(func() { tmdbSearchMovie = oldSearch })
	tmdbSearchMovie = func(_ *tmdb.Client, _ string, _ map[string]string) (*tmdb.SearchMovies, error) {
		return &tmdb.SearchMovies{}, nil
	}

	// No movie title should produce an error
	err := mockTmdb.searchByTitle(m)
	if err != ErrNoMovieTitle {
		log.Fatalf("Got %q, expected %q", err, ErrNoMovieTitle)
	}

	// Nothing to do if the id is already found
	m.Title = "Matrix"
	m.TmdbID = 12345
	err = mockTmdb.searchByTitle(m)
	if err != nil {
		log.Fatal("Search the Tmdb ID of movie with a tmdb ID should not produce an error", err)
	}
}

func TestTmdbSearchByTitle(t *testing.T) {
	m := &polochon.Movie{Title: "Matrix"}

	oldSearch := tmdbSearchMovie
	t.Cleanup(func() { tmdbSearchMovie = oldSearch })
	tmdbSearchMovie = func(_ *tmdb.Client, _ string, _ map[string]string) (*tmdb.SearchMovies, error) {
		return &tmdb.SearchMovies{
			SearchMoviesResults: &tmdb.SearchMoviesResults{
				Results: []tmdb.MovieResult{
					{Title: "The Simpsons", ID: 1000},
					{Title: "The Matrix", ID: 2000},
					{Title: "Titanic", ID: 3000},
				},
			},
		}, nil
	}

	err := mockTmdb.searchByTitle(m)
	if err != nil {
		t.Fatal(err)
	}

	if m.TmdbID != 2000 {
		t.Errorf("Failed to find tmdb id, expected 2000, got %d", m.TmdbID)
	}
}

func TestTmdbSearchByTitleNoResult(t *testing.T) {
	m := &polochon.Movie{Title: "Matrix"}

	oldSearch := tmdbSearchMovie
	t.Cleanup(func() { tmdbSearchMovie = oldSearch })
	tmdbSearchMovie = func(_ *tmdb.Client, _ string, _ map[string]string) (*tmdb.SearchMovies, error) {
		return &tmdb.SearchMovies{SearchMoviesResults: &tmdb.SearchMoviesResults{}}, nil
	}

	err := mockTmdb.searchByTitle(m)
	if err != ErrNoMovieFound {
		log.Fatalf("Got %q, expected %q", err, ErrNoMovieFound)
	}
}

func TestTmdbSearchByImdbIDArguments(t *testing.T) {
	m := polochon.NewMovie(polochon.MovieConfig{})

	oldSearch := tmdbSearchByImdbID
	t.Cleanup(func() { tmdbSearchByImdbID = oldSearch })
	tmdbSearchByImdbID = func(_ *tmdb.Client, _, _ string, _ map[string]string) (*tmdb.FindByID, error) {
		return &tmdb.FindByID{}, nil
	}

	err := mockTmdb.searchByImdbID(m)
	if err != ErrNoMovieImDBID {
		log.Fatalf("Got %q, expected %q", err, ErrNoMovieImDBID)
	}

	m.ImdbID = "tt0133093"
	m.TmdbID = 12345
	err = mockTmdb.searchByImdbID(m)
	if err != nil {
		log.Fatal("Search the Tmdb ID of movie with a tmdb ID should not produce an error", err)
	}
}

func TestTmdbSearchByImdbIDNoResults(t *testing.T) {
	m := &polochon.Movie{ImdbID: "tt0133093"}

	oldSearch := tmdbSearchByImdbID
	t.Cleanup(func() { tmdbSearchByImdbID = oldSearch })
	tmdbSearchByImdbID = func(_ *tmdb.Client, _, _ string, _ map[string]string) (*tmdb.FindByID, error) {
		return &tmdb.FindByID{}, nil
	}

	err := mockTmdb.searchByImdbID(m)
	if err != ErrNoMovieFound {
		log.Fatalf("Got %q, expected %q", err, ErrNoMovieFound)
	}
}

func TestTmdbSearchByImdbID(t *testing.T) {
	m := &polochon.Movie{ImdbID: "tt0133093"}

	oldSearch := tmdbSearchByImdbID
	t.Cleanup(func() { tmdbSearchByImdbID = oldSearch })
	tmdbSearchByImdbID = func(_ *tmdb.Client, _, _ string, _ map[string]string) (*tmdb.FindByID, error) {
		return fakeFindByID(1000), nil
	}

	err := mockTmdb.searchByImdbID(m)
	if err != nil {
		log.Fatalf("Expected no error, got %q", err)
	}

	if m.TmdbID != 1000 {
		t.Errorf("Failed to find tmdb id, expected 1000, got %d", m.TmdbID)
	}
}

func TestTmdbFailedToGetDetails(t *testing.T) {
	m := &polochon.Movie{Title: "The Matrix", ImdbID: "tt0133093"}

	oldImdbSearch, oldMovieSearch := tmdbSearchByImdbID, tmdbSearchMovie
	t.Cleanup(func() {
		tmdbSearchByImdbID = oldImdbSearch
		tmdbSearchMovie = oldMovieSearch
	})
	tmdbSearchByImdbID = func(_ *tmdb.Client, _, _ string, _ map[string]string) (*tmdb.FindByID, error) {
		return &tmdb.FindByID{}, nil
	}

	tmdbSearchMovie = func(_ *tmdb.Client, _ string, _ map[string]string) (*tmdb.SearchMovies, error) {
		return &tmdb.SearchMovies{SearchMoviesResults: &tmdb.SearchMoviesResults{}}, nil
	}

	err := mockTmdb.GetDetails(context.Background(), m)
	if err != ErrFailedToGetDetails {
		log.Fatalf("Got %q, expected %q", err, ErrFailedToGetDetails)
	}
}

func TestTmdbGetDetails(t *testing.T) {
	m := &polochon.Movie{TmdbID: 603}
	tm := &TmDB{}

	oldInfo := tmdbGetMovieInfo
	t.Cleanup(func() { tmdbGetMovieInfo = oldInfo })
	tmdbGetMovieInfo = func(_ *tmdb.Client, _ int, _ map[string]string) (*tmdb.MovieDetails, error) {
		return &tmdb.MovieDetails{
			ID:               603,
			IMDbID:           "tt0133093",
			OriginalLanguage: "en",
			OriginalTitle:    "The Matrix",
			Overview:         "Awesome plot",
			Status:           "Released",
			Tagline:          "Welcome to the Real World.",
			Title:            "The Matrix",
			Video:            false,
			VoteMetrics:      tmdb.VoteMetrics{VoteAverage: 7.599999904632568, VoteCount: 0x1086},
			Runtime:          0x88,
			ReleaseDate:      "1999-03-30",
			BackdropPath:     "/7u3pxc0K1wx32IleAkLv78MKgrw.jpg",
			Popularity:       3.1354422569274902,
			PosterPath:       "/ZPMhHXEhYB33YoTZZNNmezth0V.jpg",
			Genres: []tmdb.Genre{
				{ID: 1, Name: "Action"},
				{ID: 2, Name: "Sci-Fi"},
			},
		}, nil
	}

	err := tm.GetDetails(context.Background(), m)
	if err != nil {
		log.Fatalf("Expected no error, got %q", err)
	}

	expected := &polochon.Movie{
		ImdbID:        "tt0133093",
		OriginalTitle: "The Matrix",
		Plot:          "Awesome plot",
		Rating:        7.6,
		Runtime:       136,
		SortTitle:     "The Matrix",
		Tagline:       "Welcome to the Real World.",
		ThumbURL:      "https://image.tmdb.org/t/p/original/ZPMhHXEhYB33YoTZZNNmezth0V.jpg",
		FanartURL:     "https://image.tmdb.org/t/p/original/7u3pxc0K1wx32IleAkLv78MKgrw.jpg",
		Title:         "The Matrix",
		TmdbID:        603,
		Votes:         4230,
		Year:          1999,
		Genres:        []string{"Action", "Sci-Fi"},
	}

	if !reflect.DeepEqual(m, expected) {
		t.Errorf("Failed to get movie details\nGot: %+v\nExpected: %+v", m, expected)
	}
}

func TestTmdbGetDetailsWithoutArtwork(t *testing.T) {
	oldInfo := tmdbGetMovieInfo
	t.Cleanup(func() { tmdbGetMovieInfo = oldInfo })
	tmdbGetMovieInfo = func(_ *tmdb.Client, _ int, _ map[string]string) (*tmdb.MovieDetails, error) {
		return &tmdb.MovieDetails{Title: "The Matrix"}, nil
	}

	movie := &polochon.Movie{TmdbID: 603}
	if err := (&TmDB{}).GetDetails(context.Background(), movie); err != nil {
		t.Fatalf("get details: %v", err)
	}
	if movie.ThumbURL != "" || movie.FanartURL != "" {
		t.Fatalf("missing artwork produced URLs: thumb=%q fanart=%q", movie.ThumbURL, movie.FanartURL)
	}
}

func TestTmdbGetDetailsNilResponse(t *testing.T) {
	oldInfo := tmdbGetMovieInfo
	t.Cleanup(func() { tmdbGetMovieInfo = oldInfo })
	tmdbGetMovieInfo = func(_ *tmdb.Client, _ int, _ map[string]string) (*tmdb.MovieDetails, error) {
		return nil, nil
	}

	movie := &polochon.Movie{TmdbID: 603}
	if err := (&TmDB{}).GetDetails(context.Background(), movie); err != ErrFailedToGetDetails {
		t.Fatalf("got %v, want %v", err, ErrFailedToGetDetails)
	}
}
