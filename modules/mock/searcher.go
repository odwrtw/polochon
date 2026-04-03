package mock

import (
	"context"
	"fmt"

	polochon "github.com/odwrtw/polochon/lib"
)

// SearchMovie implements the searcher interface
func (mock *Mock) SearchMovie(_ context.Context, key string) ([]*polochon.Movie, error) {
	return []*polochon.Movie{
		{
			ImdbID: randomImdbID(),
			Title:  fmt.Sprintf("Movie %s", key),
			Plot:   "This is the plot of the movie",
		},
		{
			ImdbID: randomImdbID(),
			Title:  fmt.Sprintf("Movie almost %s", key),
			Plot:   "This is the plot of the almost movie",
		},
	}, nil
}

// SearchShow implements the searcher interface
func (mock *Mock) SearchShow(_ context.Context, key string) ([]*polochon.Show, error) {
	return []*polochon.Show{
		{
			ImdbID: randomImdbID(),
			Title:  fmt.Sprintf("Show %s", key),
			Plot:   "This is the plot of the show",
		},
		{
			ImdbID: randomImdbID(),
			Title:  fmt.Sprintf("Show almost %s", key),
			Plot:   "This is the plot of the almost show",
		},
	}, nil
}
