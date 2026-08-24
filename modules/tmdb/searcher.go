package tmdb

import (
	"context"

	polochon "github.com/odwrtw/polochon/lib"
)

// SearchMovie implements the polochon Searcher interface
func (t *TmDB) SearchMovie(_ context.Context, key string) ([]*polochon.Movie, error) {
	// We don't want porn (yet)
	options := map[string]string{
		"include_adult": "false",
	}

	// Search on tmdb
	r, err := tmdbSearchMovie(t.client, key, options)
	if err != nil {
		t.log.Debug("error while trying to find movie", "key", key)
		return nil, err
	}
	// Check if there is any results
	if len(r.Results) == 0 {
		t.log.Debug("failed to find movie", "key", key)
		return nil, ErrNoMovieFound
	}

	result := []*polochon.Movie{}
	for _, tMovie := range r.Results {
		m := polochon.NewMovie(polochon.MovieConfig{})
		m.TmdbID = int(tMovie.ID)
		err = t.getMovieDetails(m)
		if err != nil {
			t.log.Warn("error while getting tmdb movie details", "error", err)
			continue
		}
		result = append(result, m)
	}

	return result, nil
}

// SearchShow implements the polochon Searcher interface
// Not implemented
func (t *TmDB) SearchShow(_ context.Context, key string) ([]*polochon.Show, error) {
	return nil, ErrInvalidArgument
}
