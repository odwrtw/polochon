package simkl

import (
	"context"
	"strings"

	polochon "github.com/odwrtw/polochon/lib"
)

var _ polochon.Searcher = (*Simkl)(nil)

func (s *Simkl) SearchMovie(ctx context.Context, key string) ([]*polochon.Movie, error) {
	items, err := s.search(ctx, "movie", "movies", key)
	if err != nil {
		return nil, err
	}
	return moviesFromListings(items), nil
}

func (s *Simkl) SearchShow(ctx context.Context, key string) ([]*polochon.Show, error) {
	items, err := s.search(ctx, "tv", "tv", key)
	if err != nil {
		return nil, err
	}
	return showsFromListings(items), nil
}

func (s *Simkl) search(ctx context.Context, searchType, kind, key string) ([]listing, error) {
	if strings.TrimSpace(key) == "" {
		return nil, ErrInvalidArgument
	}
	if s.client == nil {
		return nil, ErrMissingClientID
	}
	items, err := s.client.search(ctx, searchType, key)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		// /search/movie returns endpoint_type "movies" (plural).
		if item.EndpointType != kind {
			return nil, ErrInvalidResponse
		}
	}
	items, err = s.usableListings(ctx, kind, items[:min(20, len(items))])
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return items, nil
}
