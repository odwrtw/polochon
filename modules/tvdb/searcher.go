package tvdb

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
)

// SearchMovie implements the polochon Searcher interface.
func (t *TvDB) SearchMovie(ctx context.Context, key string) ([]*polochon.Movie, error) {
	movies, err := t.client.search(ctx, key, "movie", 0)
	if err != nil {
		if isHTTPStatus(err, http.StatusNotFound) {
			return nil, ErrMovieNotFound
		}
		return nil, err
	}

	result := make([]*polochon.Movie, 0, len(movies))
	for _, item := range movies {
		id := imdbID(item.RemoteIDs)
		if id == "" {
			continue
		}
		movie := polochon.NewMovie(polochon.MovieConfig{})
		movie.Title = item.Name
		movie.ImdbID = id
		movie.Year, _ = strconv.Atoi(item.Year)
		if movie.Year == 0 {
			if date, err := time.Parse("2006-01-02", item.FirstAired); err == nil {
				movie.Year = date.Year()
			}
		}
		movie.Plot = item.Overview
		movie.ThumbURL = item.Image
		for _, remote := range item.RemoteIDs {
			if strings.EqualFold(remote.SourceName, "TheMovieDB.com") {
				movie.TmdbID, _ = strconv.Atoi(remote.ID)
				break
			}
		}
		result = append(result, movie)
	}
	if len(result) == 0 {
		return nil, ErrMovieNotFound
	}
	return result, nil
}

// SearchShow implements the polochon Searcher interface.
func (t *TvDB) SearchShow(ctx context.Context, key string) ([]*polochon.Show, error) {
	shows, err := t.client.searchSeries(ctx, key, 0)
	if err != nil {
		if isHTTPStatus(err, http.StatusNotFound) {
			return nil, ErrShowNotFound
		}
		return nil, err
	}

	result := make([]*polochon.Show, 0, len(shows))
	for _, item := range shows {
		id := imdbID(item.RemoteIDs)
		if id == "" {
			continue
		}
		show := polochon.NewShow(polochon.ShowConfig{})
		show.Title = item.Name
		show.TvdbID = item.ID
		show.ImdbID = id
		show.Year = showYear(item)
		show.Plot = item.Overview
		show.PosterURL = item.Image
		if date, err := time.Parse("2006-01-02", item.FirstAired); err == nil {
			show.FirstAired = &date
		}
		result = append(result, show)
	}
	if len(result) == 0 {
		return nil, ErrShowNotFound
	}
	return result, nil
}
