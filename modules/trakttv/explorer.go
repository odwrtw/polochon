package trakttv

import (
	"github.com/Sirupsen/logrus"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/trakttv"
)

// GetMovieList implements the explorer interface
func (trakt *TraktTV) GetMovieList(option polochon.ExplorerOption, log *logrus.Entry) ([]*polochon.Movie, error) {
	queryOption := trakttv.QueryOption{
		ExtendedInfos: []trakttv.ExtendedInfo{
			trakttv.ExtendedInfoMin,
		},
		Pagination: trakttv.Pagination{
			Page:  1,
			Limit: 20,
		},
	}
	// TODO: Implement better explore options
	// Right now, only return popular movies
	movies, err := trakt.client.PopularMovies(queryOption)
	if err != nil {
		return nil, err
	}
	result := []*polochon.Movie{}
	for _, movie := range movies {
		// Create the movie
		m := polochon.NewMovie(polochon.MovieConfig{})
		m.Title = movie.Title
		m.Year = movie.Year
		m.ImdbID = movie.IDs.ImDB

		// Append the movie
		result = append(result, m)
	}
	return result, nil
}

// GetShowList implements the Explorer interface
// Not implemented
func (trakt *TraktTV) GetShowList(option polochon.ExplorerOption, log *logrus.Entry) ([]*polochon.Show, error) {
	return nil, polochon.ErrNotAvailable
}

// AvailableMovieOptions implements the Explorer interface
func (trakt *TraktTV) AvailableMovieOptions() []polochon.ExplorerOption {
	return []polochon.ExplorerOption{
		polochon.ExploreByTitle,
		polochon.ExploreByYear,
		polochon.ExploreByRate,
		polochon.ExploreByDownloadCount,
		polochon.ExploreByLikeCount,
		polochon.ExploreByDateAdded,
	}
}

// AvailableShowOptions implements the Explorer interface
// Not implemented
func (trakt *TraktTV) AvailableShowOptions() []polochon.ExplorerOption {
	return []polochon.ExplorerOption{}
}
