package papi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestMovie(t *testing.T) {
	for _, test := range []struct {
		Movie               *Movie
		ExpectedURI         string
		ExpectedDownloadURL string
		ExpectedError       error
	}{
		{
			Movie:               &Movie{Movie: &polochon.Movie{ImdbID: "tt0086190"}},
			ExpectedURI:         "movies/tt0086190",
			ExpectedDownloadURL: "movies/tt0086190/download",
			ExpectedError:       nil,
		},
		{
			Movie:         &Movie{Movie: &polochon.Movie{}},
			ExpectedURI:   "",
			ExpectedError: ErrMissingMovieID,
		},
	} {
		gotURI, err := test.Movie.uri()
		if err != test.ExpectedError {
			t.Fatalf("expected error %q, got %q", test.ExpectedError, err)
		}

		if gotURI != test.ExpectedURI {
			t.Errorf("expected %q, got %q", test.ExpectedURI, gotURI)
		}

		gotDownloadURL, err := test.Movie.downloadURL()
		if err != test.ExpectedError {
			t.Fatalf("expected error %q, got %q", test.ExpectedError, err)
		}

		if gotDownloadURL != test.ExpectedDownloadURL {
			t.Errorf("expected %q, got %q", test.ExpectedDownloadURL, gotDownloadURL)
		}
	}
}

func TestGetMovies(t *testing.T) {
	var requestURI string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURI = r.RequestURI
		fmt.Fprint(w, `
			{
				"tt001": {
					"title": "title_1",
					"quality": "1080p",
					"release_group": "R1",
					"audio_codec": "AAC",
					"video_codec": "H.264",
					"container": "mkv"
				},
				"tt002": {
					"title": "title_2"
				}
			}
		`)
	}))
	defer ts.Close()

	expected := &MovieCollection{
		movies: map[string]*Movie{
			"tt001": {Movie: &polochon.Movie{
				BaseVideo: polochon.BaseVideo{
					VideoMetadata: polochon.VideoMetadata{
						Quality:      polochon.Quality1080p,
						ReleaseGroup: "R1",
						AudioCodec:   "AAC",
						VideoCodec:   "H.264",
						Container:    "mkv",
					},
				},
				ImdbID: "tt001",
				Title:  "title_1",
			}},
			"tt002": {Movie: &polochon.Movie{
				ImdbID: "tt002",
				Title:  "title_2",
			}},
		},
	}

	c, err := New(ts.URL)
	if err != nil {
		t.Fatalf("invalid endpoint: %q", err)
	}

	movies, err := c.GetMovies()
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(movies, expected) {
		t.Fatalf("expected: %#v, got %#v", expected, movies)
	}

	expectedRequestURI := "/movies"
	if requestURI != expectedRequestURI {
		t.Fatalf("expected URL %q, got %q", expectedRequestURI, requestURI)
	}
}

var movieResponse = `
{
	"imdb_id": "tt1392190",
	"original_title": "Mad Max: Fury Road",
	"plot": "Awesome plot",
	"rating": 7.3,
	"runtime": 120,
	"sort_title": "Mad Max: Fury Road",
	"tag_line": "What a Lovely Day.",
	"thumb": "https://image.tmdb.org/t/p/original/kqjL17yufvn9OVLyXYpvtyrFfak.jpg",
	"fanart": "https://image.tmdb.org/t/p/original/tbhdm8UJAb4ViCTsulYFL3lxMCd.jpg",
	"title": "Mad Max: Fury Road",
	"tmdb_id": 76341,
	"votes": 4721,
	"year": 2015,
	"torrents": null,
	"genres": ["Action", "Adventure", "Sci-Fi"]
}
`

func TestGetMovie(t *testing.T) {
	var requestURI string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURI = r.RequestURI
		fmt.Fprintf(w, "%s", movieResponse)
	}))
	defer ts.Close()

	expected := &Movie{Movie: &polochon.Movie{
		ImdbID:        "tt1392190",
		TmdbID:        76341,
		Title:         "Mad Max: Fury Road",
		OriginalTitle: "Mad Max: Fury Road",
		SortTitle:     "Mad Max: Fury Road",
		Tagline:       "What a Lovely Day.",
		Plot:          "Awesome plot",
		Year:          2015,
		Votes:         4721,
		Rating:        7.3,
		Runtime:       120,
		Thumb:         "https://image.tmdb.org/t/p/original/kqjL17yufvn9OVLyXYpvtyrFfak.jpg",
		Fanart:        "https://image.tmdb.org/t/p/original/tbhdm8UJAb4ViCTsulYFL3lxMCd.jpg",
		Genres:        []string{"Action", "Adventure", "Sci-Fi"},
	}}

	c, err := New(ts.URL)
	if err != nil {
		t.Fatalf("invalid endpoint: %q", err)
	}

	got, err := c.GetMovie("tt1392190")
	if err != nil {
		t.Fatalf("failed to get show: %q", err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected: %+v, got %+v", expected, got)
	}

	expectedRequestURI := "/movies/tt1392190"
	if requestURI != expectedRequestURI {
		t.Fatalf("expected URL %q, got %q", expectedRequestURI, requestURI)
	}
}
