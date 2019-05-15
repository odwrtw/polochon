package mock

import (
	"fmt"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// SearchMovie implements the searcher interface
func (mock *Mock) SearchMovie(key string, log *logrus.Entry) ([]*polochon.Movie, error) {
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
func (mock *Mock) SearchShow(key string, log *logrus.Entry) ([]*polochon.Show, error) {
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
