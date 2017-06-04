package tmdb

import (
	"log"
	"reflect"
	"testing"

	"github.com/odwrtw/polochon/lib"
	"github.com/ryanbradynd05/go-tmdb"
	"github.com/sirupsen/logrus"
)

// Fake TmDB pointer to run the tests
var mockLogger = logrus.New()
var mockLogEntry = logrus.NewEntry(mockLogger)
var mockTmdb = &TmDB{}

func TestTmdbInvalidMovieArgument(t *testing.T) {
	m := polochon.NewShowEpisode(polochon.ShowConfig{})
	err := mockTmdb.GetDetails(m, mockLogEntry)
	if err != ErrInvalidArgument {
		log.Fatalf("Got %q, expected %q", err, ErrInvalidArgument)
	}
}

func TestTmdbMissingArgument(t *testing.T) {
	p := &Params{}
	noKeyTmdb, err := New(p)
	if err != ErrMissingArgument {
		log.Fatalf("Got %q, expected %q", err, ErrMissingArgument)
	}
	if noKeyTmdb != nil {
		log.Fatalf("Got a non nil TmDB")
	}
}

func TestTmdbSearchByTitleArguments(t *testing.T) {
	m := polochon.NewMovie(polochon.MovieConfig{})

	tmdbSearchMovie = func(t *tmdb.TMDb, title string, options map[string]string) (*tmdb.MovieSearchResults, error) {
		return &tmdb.MovieSearchResults{}, nil
	}

	// No movie title should produce an error
	err := mockTmdb.searchByTitle(m, mockLogEntry)
	if err != ErrNoMovieTitle {
		log.Fatalf("Got %q, expected %q", err, ErrNoMovieTitle)
	}

	// Nothing to do if the id is already found
	m.Title = "Matrix"
	m.TmdbID = 12345
	err = mockTmdb.searchByTitle(m, mockLogEntry)
	if err != nil {
		log.Fatal("Search the Tmdb ID of movie with a tmdb ID should not produce an error", err)
	}
}

func TestTmdbSearchByTitle(t *testing.T) {
	m := &polochon.Movie{Title: "Matrix"}

	tmdbSearchMovie = func(t *tmdb.TMDb, title string, options map[string]string) (*tmdb.MovieSearchResults, error) {
		return &tmdb.MovieSearchResults{
			Results: []tmdb.MovieShort{
				{Title: "The Simpsons", ID: 1000},
				{Title: "The Matrix", ID: 2000},
				{Title: "Titanic", ID: 3000},
			},
		}, nil
	}

	// No movie title should produce an error
	err := mockTmdb.searchByTitle(m, mockLogEntry)
	if err != nil {
		t.Fatal(err)
	}

	if m.TmdbID != 2000 {
		t.Errorf("Failed to find tmdb id, expected 1000, got %d", m.TmdbID)
	}
}

func TestTmdbSearchByTitleNoResult(t *testing.T) {
	m := &polochon.Movie{Title: "Matrix"}

	tmdbSearchMovie = func(t *tmdb.TMDb, title string, options map[string]string) (*tmdb.MovieSearchResults, error) {
		return &tmdb.MovieSearchResults{Results: []tmdb.MovieShort{}}, nil
	}

	err := mockTmdb.searchByTitle(m, mockLogEntry)
	if err != ErrNoMovieFound {
		log.Fatalf("Got %q, expected %q", err, ErrNoMovieFound)
	}
}

func TestTmdbSearchByImdbIDArguments(t *testing.T) {
	m := polochon.NewMovie(polochon.MovieConfig{})

	tmdbSearchByImdbID = func(t *tmdb.TMDb, id, source string, options map[string]string) (*tmdb.FindResults, error) {
		return &tmdb.FindResults{}, nil
	}

	// No movie title should produce an error
	err := mockTmdb.searchByImdbID(m, mockLogEntry)
	if err != ErrNoMovieImDBID {
		log.Fatalf("Got %q, expected %q", err, ErrNoMovieImDBID)
	}

	// Nothing to do if the id is already found
	m.ImdbID = "tt0133093"
	m.TmdbID = 12345
	err = mockTmdb.searchByImdbID(m, mockLogEntry)
	if err != nil {
		log.Fatal("Search the Tmdb ID of movie with a tmdb ID should not produce an error", err)
	}
}

func TestTmdbSearchByImdbIDNoResults(t *testing.T) {
	m := &polochon.Movie{ImdbID: "tt0133093"}

	tmdbSearchByImdbID = func(t *tmdb.TMDb, id, source string, options map[string]string) (*tmdb.FindResults, error) {
		return &tmdb.FindResults{}, nil
	}

	err := mockTmdb.searchByImdbID(m, mockLogEntry)
	if err != ErrNoMovieFound {
		log.Fatalf("Got %q, expected %q", err, ErrNoMovieFound)
	}
}

func TestTmdbSearchByImdbID(t *testing.T) {
	m := &polochon.Movie{ImdbID: "tt0133093"}

	tmdbSearchByImdbID = func(t *tmdb.TMDb, id, source string, options map[string]string) (*tmdb.FindResults, error) {
		return &tmdb.FindResults{
			MovieResults: []tmdb.MovieShort{
				{Title: "The Matrix", ID: 1000},
			},
		}, nil
	}

	err := mockTmdb.searchByImdbID(m, mockLogEntry)
	if err != nil {
		log.Fatalf("Expected no error, got %q", err)
	}

	if m.TmdbID != 1000 {
		t.Errorf("Failed to find tmdb id, expected 1000, got %d", m.TmdbID)
	}
}

func TestTmdbFailedToGetDetails(t *testing.T) {
	m := &polochon.Movie{Title: "The Matrix", ImdbID: "tt0133093"}

	tmdbSearchByImdbID = func(t *tmdb.TMDb, id, source string, options map[string]string) (*tmdb.FindResults, error) {
		return &tmdb.FindResults{}, nil
	}

	tmdbSearchMovie = func(t *tmdb.TMDb, title string, options map[string]string) (*tmdb.MovieSearchResults, error) {
		return &tmdb.MovieSearchResults{Results: []tmdb.MovieShort{}}, nil
	}

	log := logrus.NewEntry(logrus.New())

	err := mockTmdb.GetDetails(m, log)
	if err != ErrFailedToGetDetails {
		log.Fatalf("Got %q, expected %q", err, ErrFailedToGetDetails)
	}
}

func TestTmdbGetDetails(t *testing.T) {
	m := &polochon.Movie{TmdbID: 603}
	tm := &TmDB{}

	tmdbGetMovieInfo = func(t *tmdb.TMDb, tmdbID int, options map[string]string) (*tmdb.Movie, error) {
		return &tmdb.Movie{
			ID:               603,
			ImdbID:           "tt0133093",
			OriginalLanguage: "en",
			OriginalTitle:    "The Matrix",
			Overview:         "Awesome plot",
			Status:           "Released",
			Tagline:          "Welcome to the Real World.",
			Title:            "The Matrix",
			Video:            false,
			VoteAverage:      7.599999904632568,
			VoteCount:        0x1086,
			Runtime:          0x88,
			ReleaseDate:      "1999-03-30",
			BackdropPath:     "/7u3pxc0K1wx32IleAkLv78MKgrw.jpg",
			Popularity:       3.1354422569274902,
			PosterPath:       "/ZPMhHXEhYB33YoTZZNNmezth0V.jpg",
			Genres: []struct {
				ID   int
				Name string
			}{
				{
					ID:   1,
					Name: "Action",
				},
				{
					ID:   2,
					Name: "Sci-Fi",
				},
			},
		}, nil
	}

	err := tm.GetDetails(m, mockLogEntry)
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
		Thumb:         "https://image.tmdb.org/t/p/original/ZPMhHXEhYB33YoTZZNNmezth0V.jpg",
		Fanart:        "https://image.tmdb.org/t/p/original/7u3pxc0K1wx32IleAkLv78MKgrw.jpg",
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
