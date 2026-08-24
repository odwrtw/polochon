package tmdb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	yaml "github.com/goccy/go-yaml"

	"github.com/agnivade/levenshtein"
	tmdb "github.com/cyruzin/golang-tmdb"
	polochon "github.com/odwrtw/polochon/lib"
)

// Make sure that the module is a detailer and a searcher
var (
	_ polochon.Detailer = (*TmDB)(nil)
	_ polochon.Searcher = (*TmDB)(nil)
)

// Register tvdb as a Detailer
func init() {
	polochon.RegisterModule(&TmDB{})
}

// Module constants
const (
	moduleName = "tmdb"
)

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
	client     *tmdb.Client
	log        *slog.Logger
	configured bool
}

// Params represents the module params
type Params struct {
	APIKey string `yaml:"apikey"`
}

// Init implements the module interface
func (t *TmDB) Init(p []byte, log *slog.Logger) error {
	if log == nil {
		log = slog.Default()
	}
	t.log = log.With("module", moduleName)

	if t.configured {
		return nil
	}

	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return err
	}

	return t.InitWithParams(params)
}

// InitWithParams configures the module
func (t *TmDB) InitWithParams(params *Params) error {
	if t.configured {
		return nil
	}

	if params == nil || params.APIKey == "" {
		return ErrMissingArgument
	}

	if t.log == nil {
		t.log = slog.Default().With("module", moduleName)
	}

	client, err := tmdb.Init(params.APIKey)
	if err != nil {
		return err
	}

	t.client = client
	t.configured = true

	return nil
}

// Name implements the Module interface
func (t *TmDB) Name() string {
	return moduleName
}

// Function to be overwritten during the tests
var tmdbSearchMovie = func(t *tmdb.Client, title string, options map[string]string) (*tmdb.SearchMovies, error) {
	return t.GetSearchMovies(title, options)
}

// SearchByTitle searches a movie by its title. It adds the tmdb id into the
// movie struct so it can get details later
func (t *TmDB) searchByTitle(m *polochon.Movie) error {
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
		t.log.Debug("Failed to find movie from imdb title", "title", m.Title)
		return ErrNoMovieFound
	}

	// Find the most accurate serie based on the levenshtein distance
	var movieShort tmdb.MovieResult
	minDistance := 100
	for _, result := range r.Results {
		d := levenshtein.ComputeDistance(m.Title, result.Title)
		if d < minDistance {
			minDistance = d
			movieShort = result
		}
	}

	m.TmdbID = int(movieShort.ID)

	t.log.Debug("Found movie from title", "title", m.Title)

	return nil
}

// Function to be overwritten during the tests
var tmdbSearchByImdbID = func(t *tmdb.Client, id, source string, options map[string]string) (*tmdb.FindByID, error) {
	if options == nil {
		options = map[string]string{}
	}
	options["external_source"] = source
	return t.GetFindByID(id, options)
}

// searchByImdbID searches on tmdb based on the imdb id
func (t *TmDB) searchByImdbID(m *polochon.Movie) error {
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
		t.log.Debug("Failed to find movie from imdb ID", "imdb_id", m.ImdbID)
		return ErrNoMovieFound
	}

	m.TmdbID = int(results.MovieResults[0].ID)

	t.log.Debug("Found movie from imdb ID", "imdb_id", m.ImdbID)

	return nil
}

// Function to be overwritten during the tests
var tmdbGetMovieInfo = func(t *tmdb.Client, tmdbID int, options map[string]string) (*tmdb.MovieDetails, error) {
	return t.GetMovieDetails(tmdbID, options)
}

// Status implements the Module interface
func (t *TmDB) Status() (polochon.ModuleStatus, error) {
	// Search for The Matrix on tmdb via imdbID
	results, err := tmdbSearchByImdbID(t.client, "tt0133093", "imdb_id", map[string]string{})
	if err != nil {
		return polochon.StatusFail, err
	}

	// Check if there is any results
	if len(results.MovieResults) == 0 {
		return polochon.StatusFail, ErrNoMovieFound
	}

	return polochon.StatusOK, nil
}

// GetDetails implements the Detailer interface
func (t *TmDB) GetDetails(_ context.Context, i any) error {
	m, ok := i.(*polochon.Movie)
	if !ok {
		return ErrInvalidArgument
	}

	// Search with imdb id
	if m.ImdbID != "" && m.TmdbID == 0 {
		err := t.searchByImdbID(m)
		if err != nil && err != ErrNoMovieFound {
			return err
		}
	}

	// Search with title
	if m.Title != "" && m.TmdbID == 0 {
		err := t.searchByTitle(m)
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
	movie.ImdbID = details.IMDbID
	movie.OriginalTitle = details.OriginalTitle
	movie.Plot = details.Overview
	movie.Rating = details.VoteAverage
	movie.Runtime = int(details.Runtime)
	movie.SortTitle = details.Title
	movie.Tagline = details.Tagline
	movie.ThumbURL = TmDBimageBaseURL + details.PosterPath
	movie.FanartURL = TmDBimageBaseURL + details.BackdropPath
	movie.Title = details.Title
	movie.Votes = int(details.VoteCount)
	movie.Year = year
	movie.Genres = genres

	return nil
}
