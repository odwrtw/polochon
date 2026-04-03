package polochon

import "context"

// Explorer is the interface explore new videos from different sources
type Explorer interface {
	Module
	AvailableMovieOptions() []string
	GetMovieList(ctx context.Context, option string) ([]*Movie, error)
	AvailableShowOptions() []string
	GetShowList(ctx context.Context, option string) ([]*Show, error)
}
