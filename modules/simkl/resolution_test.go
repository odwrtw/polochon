package simkl

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestAuthenticatedResolutionColdStartAndAmbiguousTitles(t *testing.T) {
	var lookups atomic.Int32
	s := testModule(t, Params{ClientID: "client", RefreshToken: "refresh-secret"}, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			tokenJSON(w, "access")
		case "/search/id":
			if r.Header.Get("Authorization") != "Bearer access" {
				t.Error("unauthenticated resolution")
			}
			lookups.Add(1)
			if r.URL.Query().Get("imdb") == "tt1" || r.URL.Query().Get("tmdb") == "42" {
				writeJSON(w, []record{{Type: "movie", IDs: ids{Simkl: "1"}}})
				return
			}
			writeJSON(w, []record{{Type: "movie", IDs: ids{Simkl: "1"}}, {Type: "movie", IDs: ids{Simkl: "2"}}})
		case "/movies/1":
			if r.Header.Get("Authorization") != "" {
				t.Error("credentials attached to public details")
			}
			a := catalogRecord(1, "tt1", "movie")
			a.IDs.TMDB = "42"
			writeJSON(w, a)
		case "/search/movie":
			writeJSON(w, []record{catalogRecord(3, "tt3", "movie"), catalogRecord(4, "tt4", "movie")})
		default:
			t.Errorf("unexpected %s", r.URL.Path)
			w.WriteHeader(404)
		}
	})
	ctx := context.Background()
	if err := s.GetDetails(ctx, &polochon.Movie{ImdbID: "tt1"}); err != nil {
		t.Fatal(err)
	}
	if err := s.GetDetails(ctx, &polochon.Movie{TmdbID: 42}); err != nil {
		t.Fatal(err)
	}
	if lookups.Load() != 1 {
		t.Fatal("resolved identities were not shared")
	}
	if err := s.GetDetails(ctx, &polochon.Movie{ImdbID: "tt2"}); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("ambiguous external identity: %v", err)
	}
	if err := s.GetDetails(ctx, &polochon.Movie{Title: "Example", Year: 2020}); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("ambiguous title: %v", err)
	}
	if err := s.GetDetails(ctx, &polochon.Movie{Title: "Example", Year: 2021}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("incorrect year accepted: %v", err)
	}
}

func TestConflictingEpisodeCoordinates(t *testing.T) {
	s := testModule(t, Params{}, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/tv/10" {
			writeJSON(w, catalogRecord(10, "tt10", "show"))
			return
		}
		writeJSON(w, []episodeRecord{{Season: "1", Episode: "1", Title: "First"}, {Season: "1", Episode: "1", Title: "Collision"}})
	})
	s.client.remember("tv", catalogRecord(10, "tt10", "show"))
	if _, err := s.GetShowCalendar(context.Background(), &polochon.Show{ImdbID: "tt10"}); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("coordinate collision accepted: %v", err)
	}
	for _, ep := range []episodeRecord{{Episode: "1"}, {Season: "bad", Episode: "1"}, {Season: "-1", Episode: "1"}, {Season: "1", Episode: "0"}} {
		if ep.validCoordinates() {
			t.Fatalf("accepted malformed episode: %+v", ep)
		}
	}
}

func TestThinSearchResultsHydrateIMDbFromPublicDetails(t *testing.T) {
	var detailCalls atomic.Int32
	s := testModule(t, Params{ClientID: "client", RefreshToken: "refresh-secret"}, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			tokenJSON(w, "access")
		case "/search/movie":
			if r.URL.Query().Get("limit") != "20" {
				t.Error("search hydration is not bounded")
			}
			writeJSON(w, []record{{Title: "Example", EndpointType: "movies", IDs: ids{SimklID: "1", TMDB: "42"}}, {Title: "No canonical identity", EndpointType: "movies", IDs: ids{SimklID: "2"}}, {EndpointType: "anime", IDs: ids{SimklID: "3"}}})
		case "/movies/1":
			detailCalls.Add(1)
			if r.Header.Get("Authorization") != "" {
				t.Error("search hydration leaked account token")
			}
			writeJSON(w, catalogRecord(1, "tt1", "movie"))
		case "/movies/2":
			detailCalls.Add(1)
			writeJSON(w, catalogRecord(2, "", "movie"))
		default:
			t.Errorf("unexpected or excluded catalog request %s", r.URL.Path)
			w.WriteHeader(404)
		}
	})
	for range 2 {
		results, err := s.SearchMovie(context.Background(), "Example")
		if err != nil || len(results) != 1 || results[0].ImdbID != "tt1" {
			t.Fatalf("thin search: %+v %v", results, err)
		}
	}
	if detailCalls.Load() != 2 {
		t.Fatal("search hydration did not reuse public details")
	}
}
