package trakttv

import (
	"github.com/Sirupsen/logrus"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/trakttv"
)

// Register trakttv as an Explorer
func init() {
	polochon.RegisterExplorer(moduleName, NewExplorer)
}

// NewExplorer creates a new TraktTV Explorer
func NewExplorer(p []byte) (polochon.Explorer, error) {
	return NewFromRawYaml(p)
}

// GetMovieList implements the explorer interface
func (trakt *TraktTV) GetMovieList(option string, log *logrus.Entry) ([]*polochon.Movie, error) {
	queryOption := trakttv.QueryOption{
		ExtendedInfos: []trakttv.ExtendedInfo{
			trakttv.ExtendedInfoMin,
		},
		Pagination: trakttv.Pagination{
			Page:  1,
			Limit: 40,
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
func (trakt *TraktTV) GetShowList(option string, log *logrus.Entry) ([]*polochon.Show, error) {
	return nil, polochon.ErrNotAvailable
}

// AvailableMovieOptions implements the Explorer interface
func (trakt *TraktTV) AvailableMovieOptions() []string {
	return []string{
		"popular",
	}
}

// AvailableShowOptions implements the Explorer interface
// Not implemented
func (trakt *TraktTV) AvailableShowOptions() []string {
	return []string{}
}
