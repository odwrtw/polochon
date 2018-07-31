package tmdb

import (
	"errors"
	"fmt"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/arbovm/levenshtein"
	"github.com/odwrtw/polochon/lib"
	"github.com/ryanbradynd05/go-tmdb"
	"github.com/sirupsen/logrus"
)

// Module constants
const (
	moduleName = "tmdb"
)

// Register tvdb as a Detailer
func init() {
	polochon.RegisterDetailer(moduleName, NewDetailer)
}

// API constants
const (
	TmDBimageBaseURL = "https://image.tmdb.org/t/p/original"
)

// TmDB errors
var (
	ErrInvalidArgument    = errors.New("tmdb: invalid argument")
	ErrMissingArgument    = errors.New("tmdb: missing argument")
	ErrNoMovieFound       = errors.New("tmdb: movie not found")
	ErrNoMovieTitle       = errors.New("tmdb: can not search for a movie with no title")
	ErrNoMovieImDBID      = errors.New("tmdb: can not search for a movie with no imdb")
	ErrFailedToGetDetails = errors.New("tmdb: failed to get movie details")
)

// TmDB implents the Detailer interface
type TmDB struct {
	client *tmdb.TMDb
}

// Params represents the module params
type Params struct {
	ApiKey string `yaml:"apikey"`
}

// New is an helper to avoid passing bytes
func New(p *Params) (*TmDB, error) {
	if p.ApiKey == "" {
		return nil, ErrMissingArgument
	}

	return &TmDB{
		client: tmdb.Init(tmdb.Config{ApiKey: p.ApiKey}),
	}, nil
}

// NewDetailer creates a new polochon Detailer
func NewDetailer(p []byte) (polochon.Detailer, error) {
	return NewFromRawYaml(p)
}

// NewFromRawYaml unmarshals the bytes as yaml as params and call the New
// function
func NewFromRawYaml(p []byte) (*TmDB, error) {
	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return nil, err
	}

	return New(params)
}

// Function to be overwritten during the tests
var tmdbSearchMovie = func(t *tmdb.TMDb, title string, options map[string]string) (*tmdb.MovieSearchResults, error) {
	return t.SearchMovie(title, options)
}

// SearchByTitle searches a movie by its title. It adds the tmdb id into the
// movie struct so it can get details later
func (t *TmDB) searchByTitle(m *polochon.Movie, log *logrus.Entry) error {
	// No title, no search
	if m.Title == "" {
		return ErrNoMovieTitle
	}

	// ID already found
	if m.TmdbID != 0 {
		return nil
	}

	// Add year option if given
	options := map[string]string{}
	if m.Year != 0 {
		options["year"] = fmt.Sprintf("%d", m.Year)
	}

	// Search on tmdb
	r, err := tmdbSearchMovie(t.client, m.Title, options)
	if err != nil {
		return err
	}

	// Check if there is any results
	if len(r.Results) == 0 {
		log.Debugf("Failed to find movie from imdb title %q", m.Title)
		return ErrNoMovieFound
	}

	// Find the most accurate serie based on the levenshtein distance
	var movieShort tmdb.MovieShort
	minDistance := 100
	for _, result := range r.Results {
		d := levenshtein.Distance(m.Title, result.Title)
		if d < minDistance {
			minDistance = d
			movieShort = result
		}
	}

	m.TmdbID = movieShort.ID

	log.Debugf("Found movie from title %q", m.Title)

	return nil
}

// Function to be overwritten during the tests
var tmdbSearchByImdbID = func(t *tmdb.TMDb, id, source string, options map[string]string) (*tmdb.FindResults, error) {
	return t.GetFind(id, "imdb_id", options)
}

// searchByImdbID searches on tmdb based on the imdb id
func (t *TmDB) searchByImdbID(m *polochon.Movie, log *logrus.Entry) error {
	// No imdb id, no search
	if m.ImdbID == "" {
		return ErrNoMovieImDBID
	}

	// ID already found
	if m.TmdbID != 0 {
		return nil
	}

	// Search on tmdb
	results, err := tmdbSearchByImdbID(t.client, m.ImdbID, "imdb_id", map[string]string{})
	if err != nil {
		return err
	}

	// Check if there is any results
	if len(results.MovieResults) == 0 {
		log.Debugf("Failed to find movie from imdb ID %q", m.ImdbID)
		return ErrNoMovieFound
	}

	m.TmdbID = results.MovieResults[0].ID

	log.Debugf("Found movie from imdb ID %q", m.ImdbID)

	return nil
}

// Function to be overwritten during the tests
var tmdbGetMovieInfo = func(t *tmdb.TMDb, tmdbID int, options map[string]string) (*tmdb.Movie, error) {
	return t.GetMovieInfo(tmdbID, options)
}

// Name implements the Module interface
func (t *TmDB) Name() string {
	return moduleName
}

// GetDetails implements the Detailer interface
func (t *TmDB) GetDetails(i interface{}, log *logrus.Entry) error {
	m, ok := i.(*polochon.Movie)
	if !ok {
		return ErrInvalidArgument
	}

	// Search with imdb id
	if m.ImdbID != "" && m.TmdbID == 0 {
		err := t.searchByImdbID(m, log)
		if err != nil && err != ErrNoMovieFound {
			return err
		}
	}

	// Search with title
	if m.Title != "" && m.TmdbID == 0 {
		err := t.searchByTitle(m, log)
		if err != nil && err != ErrNoMovieFound {
			return err
		}
	}

	// At this point if the tmdb id is still not found we can't update the
	// movie informations
	if m.TmdbID == 0 {
		return ErrFailedToGetDetails
	}

	// Fetch the full movie details and fill the polochon.Movie object
	err := t.getMovieDetails(m)
	if err != nil {
		return err
	}

	return nil
}

// getMovieDetails will get the movie details and fill the polochon.Movie with
// the result
func (t *TmDB) getMovieDetails(movie *polochon.Movie) error {
	// Search on tmdb
	details, err := tmdbGetMovieInfo(t.client, movie.TmdbID, map[string]string{})
	if err != nil {
		return err
	}

	// Get the year from the release date
	var year int
	if details.ReleaseDate != "" {
		date, err := time.Parse("2006-01-02", details.ReleaseDate)
		if err != nil {
			return err
		}
		year = date.Year()
	}

	// Get the movie genres
	genres := []string{}
	for _, g := range details.Genres {
		genres = append(genres, g.Name)
	}

	// Update movie details
	movie.ImdbID = details.ImdbID
	movie.OriginalTitle = details.OriginalTitle
	movie.Plot = details.Overview
	movie.Rating = details.VoteAverage
	movie.Runtime = int(details.Runtime)
	movie.SortTitle = details.Title
	movie.Tagline = details.Tagline
	movie.Thumb = TmDBimageBaseURL + details.PosterPath
	movie.Fanart = TmDBimageBaseURL + details.BackdropPath
	movie.Title = details.Title
	movie.Votes = int(details.VoteCount)
	movie.Year = year
	movie.Genres = genres

	return nil
}
