package imdb

import (
	"fmt"
	"sort"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/imdb-watchlist"
	"github.com/odwrtw/polochon/lib"
)

// Module constants
const (
	moduleName = "imdb"
)

// Register a new Subtitiler
func init() {
	polochon.RegisterWishlister(moduleName, New)
}

// New module
func New(params map[string]interface{}, log *logrus.Entry) (polochon.Wishlister, error) {
	k, ok := params["user_ids"]
	if !ok {
		return nil, fmt.Errorf("imdb: missing user ids")
	}

	ids, ok := k.([]interface{})
	if !ok {
		return nil, fmt.Errorf("imdb: user ids must be an array")
	}

	userIDs := []string{}
	for _, id := range ids {
		userID, ok := id.(string)
		if !ok {
			return nil, fmt.Errorf("imdb: user id must be a string")
		}
		userIDs = append(userIDs, userID)
	}

	return &Wishlist{
		log:     log,
		userIDs: userIDs,
	}, nil
}

// Wishlist holds the imdb wishlist
type Wishlist struct {
	log     *logrus.Entry
	userIDs []string
}

// Name implements the Module interface
func (w *Wishlist) Name() string {
	return moduleName
}

// wrapper function to be overwritten during the tests
var getMoviesFromImdb = func(userID string) (*[]string, error) {
	return imdbwatchlist.GetMovies(userID)
}

// GetMovieWishlist gets the movies wishlist
func (w *Wishlist) GetMovieWishlist() ([]*polochon.WishedMovie, error) {
	imdbIDs, err := w.getList(getMoviesFromImdb)
	if err != nil {
		return nil, err
	}

	wishedMovies := []*polochon.WishedMovie{}
	for _, imdbID := range imdbIDs {
		wishedMovies = append(wishedMovies, &polochon.WishedMovie{ImdbID: imdbID})
	}

	return wishedMovies, nil
}

// wrapper function to be overwritten during the tests
var getShowsFromImdb = func(userID string) (*[]string, error) {
	return imdbwatchlist.GetTvSeries(userID)
}

// GetShowWishlist gets the show wishlist
func (w *Wishlist) GetShowWishlist() ([]*polochon.WishedShow, error) {
	imdbIDs, err := w.getList(getShowsFromImdb)
	if err != nil {
		return nil, err
	}

	wishedShows := []*polochon.WishedShow{}
	for _, imdbID := range imdbIDs {
		wishedShows = append(wishedShows, &polochon.WishedShow{ImdbID: imdbID})
	}

	return wishedShows, nil
}

func (w *Wishlist) getList(f func(userid string) (*[]string, error)) ([]string, error) {
	var imdbIDs []string

	// Get all the ids
	for _, userID := range w.userIDs {
		ids, err := f(userID)
		if err != nil {
			return nil, err
		}

		if ids == nil {
			w.log.WithFields(logrus.Fields{
				"function": "getList",
				"userId":   userID,
			}).Info("imdb: got empty ids from imdb-watchlist")

			continue
		}

		imdbIDs = append(imdbIDs, *ids...)
	}

	uniqIds := unique(imdbIDs)
	sort.Strings(uniqIds)

	return uniqIds, nil
}

// unique returns an array of unique strings from an array of strings
func unique(strs []string) []string {
	var result []string

	t := map[string]bool{}
	for _, s := range strs {
		t[s] = true
	}
	for s := range t {
		result = append(result, s)
	}

	return result
}
