package imdb

import (
	"sort"

	"gopkg.in/yaml.v2"

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
	polochon.RegisterWishlister(moduleName, NewFromRawYaml)
}

// Params represents the module params
type Params struct {
	UserIDs []string `yaml:"user_ids"`
}

// NewFromRawYaml unmarshals the bytes as yaml as params and call the New
// function
func NewFromRawYaml(p []byte) (polochon.Wishlister, error) {
	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return nil, err
	}

	return New(params)
}

// New module
func New(params *Params) (polochon.Wishlister, error) {
	return &Wishlist{Params: params}, nil
}

// Wishlist holds the imdb wishlist
type Wishlist struct {
	*Params
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
func (w *Wishlist) GetMovieWishlist(log *logrus.Entry) ([]*polochon.WishedMovie, error) {
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
func (w *Wishlist) GetShowWishlist(log *logrus.Entry) ([]*polochon.WishedShow, error) {
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
	for _, userID := range w.UserIDs {
		ids, err := f(userID)
		if err != nil {
			return nil, err
		}

		if ids == nil {
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
