package polochon

import "context"

// Searcher is the interface to search shows or movies from different sources
type Searcher interface {
	Module
	SearchMovie(ctx context.Context, key string) ([]*Movie, error)
	SearchShow(ctx context.Context, key string) ([]*Show, error)
}
