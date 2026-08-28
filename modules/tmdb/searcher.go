package tmdb

import (
	"context"

	polochon "github.com/odwrtw/polochon/lib"
)

// SearchMovie implements the polochon Searcher interface. The results stay
// lightweight: each item carries its TmdbID plus the fields already provided
// by the search response, and full details are fetched only after selection.
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
	if r == nil || r.SearchMoviesResults == nil || len(r.Results) == 0 {
		t.log.Debug("failed to find movie", "key", key)
		return nil, ErrNoMovieFound
	}

	result := make([]*polochon.Movie, 0, len(r.Results))
	for _, tMovie := range r.Results {
		result = append(result, mapMovieListItem(
			tMovie.ID,
			tMovie.Title,
			tMovie.Overview,
			tMovie.ReleaseDate,
			tMovie.PosterPath,
			tMovie.BackdropPath,
			tMovie.VoteAverage,
		))
	}

	return result, nil
}

// SearchShow implements the polochon Searcher interface. The results stay
// lightweight: each item carries its TmdbID plus the fields already provided
// by the search response, and full details are fetched only after selection.
func (t *TmDB) SearchShow(_ context.Context, key string) ([]*polochon.Show, error) {
	if key == "" {
		return nil, ErrNoShowTitle
	}

	// Search on tmdb
	r, err := tmdbSearchTVShow(t.client, key, map[string]string{})
	if err != nil {
		t.log.Debug("error while trying to find show", "key", key)
		return nil, err
	}
	// Check if there is any results
	if r == nil || r.SearchTVShowsResults == nil || len(r.Results) == 0 {
		t.log.Debug("failed to find show", "key", key)
		return nil, ErrNoShowFound
	}

	result := make([]*polochon.Show, 0, len(r.Results))
	for _, tShow := range r.Results {
		result = append(result, mapShowListItem(
			tShow.ID,
			tShow.Name,
			tShow.Overview,
			tShow.FirstAirDate,
			tShow.PosterPath,
			tShow.BackdropPath,
			tShow.VoteAverage,
		))
	}

	return result, nil
}
