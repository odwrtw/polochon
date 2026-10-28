package simkl

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestWishlistSnapshotSharedImmutableAndRestartFreshness(t *testing.T) {
	var activities, movies, shows atomic.Int32
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			tokenJSON(w, "access")
			return
		}
		if r.Method != "GET" {
			t.Error("unexpected account write")
		}
		if r.Header.Get("Authorization") != "Bearer access" {
			t.Error("missing account token")
		}
		switch r.URL.Path {
		case "/sync/activities":
			activities.Add(1)
			writeJSON(w, map[string]string{"all": "2026-01-01T00:00:00Z"})
		case "/sync/all-items/movies/plantowatch":
			if activities.Load() == 0 {
				t.Error("snapshot fetched before activities")
			}
			movies.Add(1)
			items := []wishlistItem{{Status: "plantowatch", Movie: catalogRecord(1, "tt1", "movie")}, {Status: "completed", Movie: catalogRecord(2, "tt2", "movie")}, {Movie: catalogRecord(3, "", "movie")}}
			writeJSON(w, wishlistResponse{Movies: items})
		case "/sync/all-items/shows/plantowatch":
			shows.Add(1)
			writeJSON(w, wishlistResponse{Shows: []wishlistItem{{Show: catalogRecord(10, "tt10", "show")}, {Show: catalogRecord(11, "tt11", "show")}, {Show: catalogRecord(12, "tt11", "show")}}})
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}
	p := Params{ClientID: "client", RefreshToken: "refresh-secret"}
	s := testModule(t, p, handler)
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			if _, err := s.GetMovieWishlist(context.Background()); err != nil {
				t.Error(err)
			}
		})
		wg.Go(func() {
			if _, err := s.GetShowWishlist(context.Background()); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if activities.Load() != 1 || movies.Load() != 1 || shows.Load() != 1 {
		t.Fatal("scheduled reads triggered remote polling")
	}
	ml, err := s.GetMovieWishlist(context.Background())
	if err != nil || len(ml) != 1 || ml[0].ImdbID != "tt1" {
		t.Fatalf("movies: %+v %v", ml, err)
	}
	sl, err := s.GetShowWishlist(context.Background())
	if err != nil || len(sl) != 1 || sl[0].ImdbID != "tt10" || sl[0].Season != 1 || sl[0].Episode != 1 {
		t.Fatalf("shows: %+v %v", sl, err)
	}
	ml[0].ImdbID = "modified"
	sl[0].Season = 100
	ml, _ = s.GetMovieWishlist(context.Background())
	sl, _ = s.GetShowWishlist(context.Background())
	if ml[0].ImdbID != "tt1" || sl[0].Season != 1 {
		t.Fatal("caller mutated shared snapshot")
	}
	// Existing quality preferences still apply through the real wishlist consumer.
	w := polochon.NewWishlist(polochon.WishlistConfig{Wishlisters: []polochon.Wishlister{s}, MovieDefaultQualities: []polochon.Quality{"1080p"}, ShowDefaultQualities: []polochon.Quality{"720p"}}, s.log)
	if err = w.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(w.Movies) != 1 || len(w.Movies[0].Qualities) != 1 || w.Movies[0].Qualities[0] != "1080p" || w.Shows[0].Qualities[0] != "720p" {
		t.Fatal("existing wishlist quality behavior changed")
	}
	fresh := testModule(t, p, handler)
	if _, err = fresh.GetMovieWishlist(context.Background()); err != nil {
		t.Fatal(err)
	}
	if activities.Load() != 2 || movies.Load() != 2 || shows.Load() != 2 {
		t.Fatal("cold start did not refresh snapshot")
	}
}

func TestWishlistPartialFailureBackoffAndLargeSnapshot(t *testing.T) {
	var showCalls atomic.Int32
	now := time.Now()
	s := testModule(t, Params{ClientID: "client", RefreshToken: "refresh-secret"}, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			tokenJSON(w, "access")
		case "/sync/activities":
			writeJSON(w, map[string]string{"all": "now"})
		case "/sync/all-items/movies/plantowatch":
			items := make([]wishlistItem, 600)
			for i := range items {
				items[i] = wishlistItem{Movie: catalogRecord(i+1, "tt"+strconv.Itoa(i+1), "movie")}
			}
			writeJSON(w, wishlistResponse{Movies: items})
		case "/sync/all-items/shows/plantowatch":
			if showCalls.Add(1) == 1 {
				w.WriteHeader(400)
				writeJSON(w, map[string]string{"error": "max_items"})
				return
			}
			writeJSON(w, wishlistResponse{})
		}
	})
	s.client.now = func() time.Time { return now }
	if _, err := s.GetMovieWishlist(context.Background()); err == nil {
		t.Fatal("partial snapshot published")
	}
	if s.snapshot != nil {
		t.Fatal("saved failed snapshot")
	}
	if _, err := s.GetShowWishlist(context.Background()); err == nil || showCalls.Load() != 1 {
		t.Fatal("failed snapshot has no backoff")
	}
	now = now.Add(time.Minute)
	items, err := s.GetMovieWishlist(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 600 {
		t.Fatalf("wishlist silently truncated: %d", len(items))
	}
	if _, err = s.client.known("movies", "imdb", "tt600"); err != nil {
		t.Fatal(err)
	}
	public := &Simkl{}
	if _, err = public.GetMovieWishlist(context.Background()); !errors.Is(err, ErrConnectionRequired) {
		t.Fatal(err)
	}
}
