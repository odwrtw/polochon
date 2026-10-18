package trakttv

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/odwrtw/trakttv"
)

func TestAnticipatedLists(t *testing.T) {
	for _, tc := range []struct {
		name     string
		resource string
	}{
		{name: "movies", resource: "movie"},
		{name: "shows", resource: "show"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if want := "/" + tc.name + "/anticipated"; r.URL.Path != want {
					t.Errorf("request path = %q, want %q", r.URL.Path, want)
				}
				w.Header().Set("Content-Type", "application/json")
				if _, err := fmt.Fprintf(w, `[{"list_count":42,%q:{"title":"Anticipated title","year":2027,"ids":{"imdb":"tt1234567"}}}]`, tc.resource); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)

			client := trakttv.New("test-client-id")
			client.Endpoint = server.URL
			client.HTTPClient = server.Client()
			trakt := &TraktTV{client: client}

			var title, imdbID string
			var year int
			switch tc.resource {
			case "movie":
				movies, err := trakt.GetMovieList(context.Background(), "anticipated")
				if err != nil {
					t.Fatal(err)
				}
				if len(movies) != 1 {
					t.Fatalf("got %d movies, want 1", len(movies))
				}
				title, year, imdbID = movies[0].Title, movies[0].Year, movies[0].ImdbID
			case "show":
				shows, err := trakt.GetShowList(context.Background(), "anticipated")
				if err != nil {
					t.Fatal(err)
				}
				if len(shows) != 1 {
					t.Fatalf("got %d shows, want 1", len(shows))
				}
				title, year, imdbID = shows[0].Title, shows[0].Year, shows[0].ImdbID
			}

			if title != "Anticipated title" || year != 2027 || imdbID != "tt1234567" {
				t.Errorf("got title=%q year=%d imdb=%q, want Anticipated title, 2027, tt1234567", title, year, imdbID)
			}
		})
	}
}
