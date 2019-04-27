package mock

import (
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// GetMovieWishlist implements the wishlister interface
func (mock *Mock) GetMovieWishlist(*logrus.Entry) ([]*polochon.WishedMovie, error) {
	return nil, nil
}

// GetShowWishlist implements the wishlister interface
func (mock *Mock) GetShowWishlist(*logrus.Entry) ([]*polochon.WishedShow, error) {
	return nil, nil
}
