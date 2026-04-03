package mock

import (
	"context"

	polochon "github.com/odwrtw/polochon/lib"
)

// GetMovieWishlist implements the wishlister interface
func (mock *Mock) GetMovieWishlist(_ context.Context) ([]*polochon.WishedMovie, error) {
	return nil, nil
}

// GetShowWishlist implements the wishlister interface
func (mock *Mock) GetShowWishlist(_ context.Context) ([]*polochon.WishedShow, error) {
	return nil, nil
}
