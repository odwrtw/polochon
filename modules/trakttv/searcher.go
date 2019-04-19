package trakttv

import (
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/trakttv"
	"github.com/sirupsen/logrus"
)

// SearchMovie implements the polochon Searcher interface
func (trakt *TraktTV) SearchMovie(key string, log *logrus.Entry) ([]*polochon.Movie, error) {
	searchQuery := trakttv.SearchQuery{
		Type:  trakttv.TypeMovie,
		Field: trakttv.FieldTitle,
		Query: key,
	}
	queryOption := trakttv.QueryOption{
		Pagination: trakttv.Pagination{
			Page:  1,
			Limit: 20,
		},
	}

	traktMovies, err := trakt.client.Search(searchQuery, queryOption)
	if err != nil {
		return nil, err
	}

	// Check if there is any results
	if len(traktMovies) == 0 {
		log.Debugf("failed to find movie with %q", key)
		return nil, ErrNotFound
	}

	result := []*polochon.Movie{}
	for _, tMovie := range traktMovies {
		// Skip movies without nil score
		if tMovie.Score == 0 {
			continue
		}
		// Skip movies without imdb ID
		if tMovie.Movie.IDs.ImDB == "" {
			continue
		}
		m := polochon.NewMovie(polochon.MovieConfig{})
		m.ImdbID = tMovie.Movie.IDs.ImDB
		m.TmdbID = tMovie.Movie.IDs.TmDB
		m.Title = tMovie.Movie.Title
		m.Year = tMovie.Movie.Year
		result = append(result, m)
	}

	return result, nil
}

// SearchShow implements the polochon Searcher interface
func (trakt *TraktTV) SearchShow(key string, log *logrus.Entry) ([]*polochon.Show, error) {
	searchQuery := trakttv.SearchQuery{
		Type:  trakttv.TypeShow,
		Field: trakttv.FieldTitle,
		Query: key,
	}
	queryOption := trakttv.QueryOption{
		Pagination: trakttv.Pagination{
			Page:  1,
			Limit: 20,
		},
	}

	traktShows, err := trakt.client.Search(searchQuery, queryOption)
	if err != nil {
		return nil, err
	}

	// Check if there is any results
	if len(traktShows) == 0 {
		log.Debugf("failed to find shows with %q", key)
		return nil, ErrNotFound
	}

	result := []*polochon.Show{}
	for _, tShow := range traktShows {
		// Skip shows without nil score
		if tShow.Score == 0 {
			continue
		}
		// Skip shows without imdb ID
		if tShow.Show.IDs.ImDB == "" {
			continue
		}
		s := polochon.NewShow(polochon.ShowConfig{})
		s.ImdbID = tShow.Show.IDs.ImDB
		s.TvdbID = tShow.Show.IDs.TvDB
		s.Title = tShow.Show.Title
		s.Year = tShow.Show.Year
		result = append(result, s)
	}

	return result, nil
}
