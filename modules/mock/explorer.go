package mock

import (
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// AvailableMovieOptions implements the explorer interface
func (mock *Mock) AvailableMovieOptions() []string {
	return []string{"byWTF", "byCoolness"}
}

// AvailableShowOptions implements the explorer interface
func (mock *Mock) AvailableShowOptions() []string {
	return []string{"byWTF", "byCoolness"}
}

// GetMovieList implements the explorer interface
func (mock *Mock) GetMovieList(option string, log *logrus.Entry) ([]*polochon.Movie, error) {
	var movies []*polochon.Movie
	for i := 1; i <= 20; i++ {
		movies = append(movies, &polochon.Movie{
			ImdbID: randomImdbID(),
		})
	}
	return movies, nil
}

// GetShowList implements the explorer interface
func (mock *Mock) GetShowList(option string, log *logrus.Entry) ([]*polochon.Show, error) {
	var shows []*polochon.Show
	for i := 1; i <= 20; i++ {
		shows = append(shows, &polochon.Show{
			ImdbID: randomImdbID(),
		})
	}
	return shows, nil
}
