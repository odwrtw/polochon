package mock

import (
	"context"

	polochon "github.com/odwrtw/polochon/lib"
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
func (mock *Mock) GetMovieList(_ context.Context, option string) ([]*polochon.Movie, error) {
	var movies []*polochon.Movie
	for i := 1; i <= 20; i++ {
		movies = append(movies, &polochon.Movie{
			ImdbID: randomImdbID(),
		})
	}
	return movies, nil
}

// GetShowList implements the explorer interface
func (mock *Mock) GetShowList(_ context.Context, option string) ([]*polochon.Show, error) {
	var shows []*polochon.Show
	for i := 1; i <= 20; i++ {
		shows = append(shows, &polochon.Show{
			ImdbID: randomImdbID(),
		})
	}
	return shows, nil
}
