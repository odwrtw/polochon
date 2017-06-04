package tmdb

import (
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// Register tmdb as a Searcher
func init() {
	polochon.RegisterSearcher(moduleName, NewSearcher)
}

// NewSearcher creates a new Tmdb Searcher
func NewSearcher(p []byte) (polochon.Searcher, error) {
	return NewFromRawYaml(p)
}

// SearchMovie implements the polochon Searcher interface
func (t *TmDB) SearchMovie(key string, log *logrus.Entry) ([]*polochon.Movie, error) {
	// We don't want porn (yet)
	options := map[string]string{
		"include_adult": "false",
	}

	// Search on tmdb
	r, err := tmdbSearchMovie(t.client, key, options)
	if err != nil {
		log.Debugf("error while trying to find movie with %q", key)
		return nil, err
	}
	// Check if there is any results
	if len(r.Results) == 0 {
		log.Debugf("failed to find movie with %q", key)
		return nil, ErrNoMovieFound
	}

	result := []*polochon.Movie{}
	for _, tMovie := range r.Results {
		m := polochon.NewMovie(polochon.MovieConfig{})
		m.TmdbID = tMovie.ID
		err = t.getMovieDetails(m)
		if err != nil {
			log.Warnf("error while getting tmdb movie details %q", err)
			continue
		}
		result = append(result, m)
	}

	return result, nil
}

// SearchShow implements the polochon Searcher interface
// Not implemented
func (t *TmDB) SearchShow(key string, log *logrus.Entry) ([]*polochon.Show, error) {
	return nil, ErrInvalidArgument
}
