package simkl

import (
	"context"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
)

type wishlistItem struct {
	Status string `json:"status"`
	Movie  record `json:"movie"`
	Show   record `json:"show"`
}
type wishlistResponse struct {
	Movies []wishlistItem `json:"movies"`
	Shows  []wishlistItem `json:"shows"`
}
type wishlistSnapshot struct {
	movies []*polochon.WishedMovie
	shows  []*polochon.WishedShow
	at     time.Time
}

func (s *Simkl) wishlist(ctx context.Context) (*wishlistSnapshot, error) {
	c, err := s.ready()
	if err != nil {
		return nil, err
	}
	if err = acquire(ctx, s.snapshotGate); err != nil {
		return nil, err
	}
	defer func() { <-s.snapshotGate }()
	if s.snapshot != nil {
		return s.snapshot, nil
	}
	if c.now().Before(s.snapshotRetry) {
		return nil, s.snapshotErr
	}
	result, err := s.loadWishlist(ctx)
	if err != nil {
		if ctx.Err() == nil {
			s.snapshotErr = err
			s.snapshotRetry = c.now().Add(time.Minute)
		}
		return nil, err
	}
	s.snapshot = result
	s.log.Info("loaded Plan-to-Watch snapshot; restart to refresh", "movies", len(result.movies), "shows", len(result.shows), "synced_at", result.at)
	return result, nil
}

func (s *Simkl) loadWishlist(ctx context.Context) (*wishlistSnapshot, error) {
	c := s.client
	var activities map[string]any
	if err := c.get(ctx, c.api+"/sync/activities", true, 0, false, &activities); err != nil {
		return nil, err
	}
	// Split the initial read by both type and status to avoid full-library limits.
	var movies, shows wishlistResponse
	if err := c.get(ctx, c.api+"/sync/all-items/movies/plantowatch?extended=ids_only", true, 0, false, &movies); err != nil {
		return nil, err
	}
	if err := c.get(ctx, c.api+"/sync/all-items/shows/plantowatch?extended=ids_only", true, 0, false, &shows); err != nil {
		return nil, err
	}
	movieRecords := make([]record, 0, len(movies.Movies))
	showRecords := make([]record, 0, len(shows.Shows))
	for _, v := range movies.Movies {
		if v.Status == "" || v.Status == "plantowatch" {
			movieRecords = append(movieRecords, v.Movie)
		}
	}
	for _, v := range shows.Shows {
		if v.Status == "" || v.Status == "plantowatch" {
			showRecords = append(showRecords, v.Show)
		}
	}
	snapshot := &wishlistSnapshot{movies: []*polochon.WishedMovie{}, shows: []*polochon.WishedShow{}, at: c.now()}
	// addressable rejects conflicting canonical identities before IMDb-based merging.
	for _, r := range s.addressable("movies", movieRecords) {
		snapshot.movies = append(snapshot.movies, &polochon.WishedMovie{ImdbID: r.IDs.IMDb})
	}
	for _, r := range s.addressable("tv", showRecords) {
		snapshot.shows = append(snapshot.shows, &polochon.WishedShow{ImdbID: r.IDs.IMDb, Season: 1, Episode: 1})
	}
	return snapshot, nil
}

func (s *Simkl) GetMovieWishlist(ctx context.Context) ([]*polochon.WishedMovie, error) {
	snapshot, err := s.wishlist(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*polochon.WishedMovie, 0, len(snapshot.movies))
	for _, v := range snapshot.movies {
		copy := *v
		out = append(out, &copy)
	}
	return out, nil
}
func (s *Simkl) GetShowWishlist(ctx context.Context) ([]*polochon.WishedShow, error) {
	snapshot, err := s.wishlist(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*polochon.WishedShow, 0, len(snapshot.shows))
	for _, v := range snapshot.shows {
		copy := *v
		out = append(out, &copy)
	}
	return out, nil
}
