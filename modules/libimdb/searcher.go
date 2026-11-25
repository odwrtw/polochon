package libimdb

import (
	"context"

	imdb "github.com/odwrtw/libimdb"

	polochon "github.com/odwrtw/polochon/lib"
)

// SearchMovie returns partial catalog models for suggestions, never automatic
// identities. The Searcher interface does not expose scoring or truncation.
func (c *Catalog) SearchMovie(ctx context.Context, key string) ([]*polochon.Movie, error) {
	result, err := c.candidates(ctx, key, imdb.Movie)
	if err != nil {
		return nil, err
	}
	movies := make([]*polochon.Movie, 0, len(result.Matches))
	for _, match := range result.Matches {
		title := match.Title
		movie := &polochon.Movie{ImdbID: title.ID, Title: title.PrimaryTitle, OriginalTitle: title.OriginalTitle, Genres: title.Genres}
		if title.StartYear != nil {
			movie.Year = *title.StartYear
		}
		if title.RuntimeMinutes != nil {
			movie.Runtime = *title.RuntimeMinutes
		}
		if title.Rating != nil {
			movie.Rating = float32(*title.Rating)
		}
		if title.Votes != nil {
			// Consumer platforms (Linux amd64/arm64) have 64-bit ints.
			movie.Votes = int(*title.Votes)
		}
		movies = append(movies, movie)
	}
	return movies, nil
}

// SearchShow returns series facts only; episodes, artwork and air dates need
// provider enrichment. Empty scalar fields represent unavailable catalog facts.
func (c *Catalog) SearchShow(ctx context.Context, key string) ([]*polochon.Show, error) {
	result, err := c.candidates(ctx, key, imdb.Show)
	if err != nil {
		return nil, err
	}
	shows := make([]*polochon.Show, 0, len(result.Matches))
	for _, match := range result.Matches {
		title := match.Title
		show := &polochon.Show{ImdbID: title.ID, Title: title.PrimaryTitle}
		if title.StartYear != nil {
			show.Year = *title.StartYear
		}
		if title.Rating != nil {
			show.Rating = float32(*title.Rating)
		}
		shows = append(shows, show)
	}
	return shows, nil
}

func (c *Catalog) candidates(ctx context.Context, key string, kind imdb.Kind) (imdb.SearchResult, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	catalog, err := c.open(ctx)
	if err != nil {
		return imdb.SearchResult{}, err
	}
	defer func() { _ = catalog.Close() }()
	// Library defaults bound both the retrieval budget and output size.
	return catalog.Candidates(ctx, imdb.Query{Title: key, Kind: kind}, imdb.SearchOptions{})
}
