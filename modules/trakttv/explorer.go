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
	var movies []*trakttv.Movie
	var err error
	switch option {
	case "popular":
		movies, err = trakt.client.PopularMovies(queryOption)
		if err != nil {
			return nil, err
		}
	case "trending":
		trendingMovies, err := trakt.client.TrendingMovies(queryOption)
		if err != nil {
			return nil, err
		}
		for _, tMovie := range trendingMovies {
			movies = append(movies, &tMovie.Movie)
		}
	case "anticicpated":
		anticipatedMovies, err := trakt.client.AnticipatedMovies(queryOption)
		if err != nil {
			return nil, err
		}
		for _, aMovie := range anticipatedMovies {
			movies = append(movies, &aMovie.Movie)
		}
	case "boxoffice":
		boxOfficeMovies, err := trakt.client.BoxOfficeMovies(queryOption)
		if err != nil {
			return nil, err
		}
		for _, bMovie := range boxOfficeMovies {
			movies = append(movies, &bMovie.Movie)
		}
	case "played":
		playedMovies, err := trakt.client.PlayedMovies(queryOption)
		if err != nil {
			return nil, err
		}
		for _, pMovie := range playedMovies {
			movies = append(movies, &pMovie.Movie)
		}
	case "watched":
		watchedMovies, err := trakt.client.WatchedMovies(queryOption)
		if err != nil {
			return nil, err
		}
		for _, wMovie := range watchedMovies {
			movies = append(movies, &wMovie.Movie)
		}
	case "collected":
		collectedMovies, err := trakt.client.CollectedMovies(queryOption)
		if err != nil {
			return nil, err
		}
		for _, cMovie := range collectedMovies {
			movies = append(movies, &cMovie.Movie)
		}
	default:
		return nil, ErrInvalidArgument
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
	queryOption := trakttv.QueryOption{
		ExtendedInfos: []trakttv.ExtendedInfo{
			trakttv.ExtendedInfoMin,
		},
		Pagination: trakttv.Pagination{
			Page:  1,
			Limit: 40,
		},
	}
	var shows []*trakttv.Show
	var err error
	switch option {
	case "popular":
		shows, err = trakt.client.PopularShows(queryOption)
		if err != nil {
			return nil, err
		}
	case "trending":
		trendingShows, err := trakt.client.TrendingShows(queryOption)
		if err != nil {
			return nil, err
		}
		for _, tShow := range trendingShows {
			shows = append(shows, &tShow.Show)
		}
	case "anticicpated":
		anticipatedShows, err := trakt.client.AnticipatedShows(queryOption)
		if err != nil {
			return nil, err
		}
		for _, aShow := range anticipatedShows {
			shows = append(shows, &aShow.Show)
		}
	case "played":
		playedShows, err := trakt.client.PlayedShows(queryOption)
		if err != nil {
			return nil, err
		}
		for _, pShow := range playedShows {
			shows = append(shows, &pShow.Show)
		}
	case "watched":
		watchedShows, err := trakt.client.WatchedShows(queryOption)
		if err != nil {
			return nil, err
		}
		for _, wShow := range watchedShows {
			shows = append(shows, &wShow.Show)
		}
	case "collected":
		collectedShows, err := trakt.client.CollectedShows(queryOption)
		if err != nil {
			return nil, err
		}
		for _, cShow := range collectedShows {
			shows = append(shows, &cShow.Show)
		}
	default:
		return nil, ErrInvalidArgument
	}
	result := []*polochon.Show{}
	for _, show := range shows {
		// Create the show
		m := polochon.NewShow(polochon.ShowConfig{})
		m.Title = show.Title
		m.Year = show.Year
		m.ImdbID = show.IDs.ImDB

		// Append the show
		result = append(result, m)
	}
	return result, nil
}

// AvailableMovieOptions implements the Explorer interface
func (trakt *TraktTV) AvailableMovieOptions() []string {
	return []string{
		"popular",
		"trending",
		"boxoffice",
		"anticipated",
		"watched",
		"collected",
		"played",
	}
}

// AvailableShowOptions implements the Explorer interface
// Not implemented
func (trakt *TraktTV) AvailableShowOptions() []string {
	return []string{
		"popular",
		"trending",
		"anticipated",
		"watched",
		"collected",
		"played",
	}
}
