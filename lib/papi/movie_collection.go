package papi

import "sort"

// MovieCollection represent a collection of movies
type MovieCollection struct {
	movies map[string]*Movie
}

// NewMovieCollection returns a new empty collection of movies
func NewMovieCollection() *MovieCollection {
	return &MovieCollection{
		movies: map[string]*Movie{},
	}
}

// List return a list of movies
func (m *MovieCollection) List() []*Movie {
	movies := make([]*Movie, len(m.movies))

	i := 0
	for _, v := range m.movies {
		movies[i] = v
		i++
	}

	// Sort the slice by Imdb ID
	sort.Slice(movies, func(i, j int) bool {
		return movies[i].ImdbID < movies[j].ImdbID
	})

	return movies
}

// Has checks if a movie is in a collection
func (m *MovieCollection) Has(imdbID string) (*Movie, bool) {
	movie, ok := m.movies[imdbID]
	return movie, ok
}

// Add adds a movie to the collection
func (m *MovieCollection) Add(movie *Movie) error {
	if movie == nil || movie.Movie == nil {
		return ErrMissingMovie
	}

	if movie.ImdbID == "" {
		return ErrMissingMovieID
	}

	m.movies[movie.ImdbID] = movie

	return nil
}
