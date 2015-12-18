package tmdb

import (
	"errors"
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/arbovm/levenshtein"
	"github.com/odwrtw/polochon/lib"
	"github.com/ryanbradynd05/go-tmdb"
)

// Module constants
const (
	moduleName = "tmdb"
)

// Register tvdb as a Detailer
func init() {
	polochon.RegisterDetailer(moduleName, NewFromRawYaml)
}

// API constants
const (
	TmDBAPIKey       = "9b939aee0aaafc12a65bf448e4af9543"
	TmDBimageBaseURL = "https://image.tmdb.org/t/p/original"
)

// TmDB errors
var (
	ErrInvalidArgument    = errors.New("tmdb: invalid argument")
	ErrNoMovieFound       = errors.New("tmdb: movie not found")
	ErrNoMovieTitle       = errors.New("tmdb: can not search for a movie with no title")
	ErrNoMovieImDBID      = errors.New("tmdb: can not search for a movie with no imdb")
	ErrFailedToGetDetails = errors.New("tmdb: failed to get movie details")
)

// TmDB implents the Detailer interface
type TmDB struct{}

// New is an helper to avoid passing bytes
func New() (polochon.Detailer, error) {
	return &TmDB{}, nil
}

// NewFromRawYaml unmarshals the bytes as yaml as params and call the New
// function
func NewFromRawYaml(p []byte) (polochon.Detailer, error) {
	return New()
}

// Function to be overwritten during the tests
var tmdbSearchMovie = func(title string, options map[string]string) (*tmdb.MovieSearchResults, error) {
	t := tmdb.Init(TmDBAPIKey)
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
	r, err := tmdbSearchMovie(m.Title, options)
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
var tmdbSearchByImdbID = func(id, source string, options map[string]string) (*tmdb.FindResults, error) {
	t := tmdb.Init(TmDBAPIKey)
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
	results, err := tmdbSearchByImdbID(m.ImdbID, "imdb_id", map[string]string{})
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
var tmdbGetMovieInfo = func(tmdbID int, options map[string]string) (*tmdb.Movie, error) {
	t := tmdb.Init(TmDBAPIKey)
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

	// Search on tmdb
	details, err := tmdbGetMovieInfo(m.TmdbID, map[string]string{})
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

	// Update movie details
	m.ImdbID = details.ImdbID
	m.OriginalTitle = details.OriginalTitle
	m.Plot = details.Overview
	m.Rating = details.VoteAverage
	m.Runtime = int(details.Runtime)
	m.SortTitle = details.Title
	m.Tagline = details.Tagline
	m.Thumb = TmDBimageBaseURL + details.PosterPath
	m.Fanart = TmDBimageBaseURL + details.BackdropPath
	m.Title = details.Title
	m.Votes = int(details.VoteCount)
	m.Year = year

	return nil
}
