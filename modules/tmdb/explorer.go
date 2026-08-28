package tmdb

import (
	"context"
	"errors"
	"time"

	tmdb "github.com/cyruzin/golang-tmdb"
	polochon "github.com/odwrtw/polochon/lib"
)

// Explorer options
const (
	movieOptionPopular    = "popular"
	movieOptionTopRated   = "top_rated"
	movieOptionUpcoming   = "upcoming"
	movieOptionNowPlaying = "now_playing"

	showOptionPopular     = "popular"
	showOptionTopRated    = "top_rated"
	showOptionAiringToday = "airing_today"
	showOptionOnTheAir    = "on_the_air"
)

// ErrInvalidOption is returned when an explorer option is not a native TMDB
// category.
var ErrInvalidOption = errors.New("tmdb: invalid explorer option")

// These wrappers keep the list calls replaceable by unit tests, as the search,
// detail and external-ID calls are in tmdb.go and detailer.go.
var (
	tmdbGetMoviePopular = func(t *tmdb.Client, options map[string]string) (*tmdb.MoviePopular, error) {
		return t.GetMoviePopular(options)
	}
	tmdbGetMovieTopRated = func(t *tmdb.Client, options map[string]string) (*tmdb.MovieTopRated, error) {
		return t.GetMovieTopRated(options)
	}
	tmdbGetMovieUpcoming = func(t *tmdb.Client, options map[string]string) (*tmdb.MovieUpcoming, error) {
		return t.GetMovieUpcoming(options)
	}
	tmdbGetMovieNowPlaying = func(t *tmdb.Client, options map[string]string) (*tmdb.MovieNowPlaying, error) {
		return t.GetMovieNowPlaying(options)
	}
	tmdbGetTVAiringToday = func(t *tmdb.Client, options map[string]string) (*tmdb.TVAiringToday, error) {
		return t.GetTVAiringToday(options)
	}
	tmdbGetTVOnTheAir = func(t *tmdb.Client, options map[string]string) (*tmdb.TVOnTheAir, error) {
		return t.GetTVOnTheAir(options)
	}
	tmdbGetTVPopular = func(t *tmdb.Client, options map[string]string) (*tmdb.TVPopular, error) {
		return t.GetTVPopular(options)
	}
	tmdbGetTVTopRated = func(t *tmdb.Client, options map[string]string) (*tmdb.TVTopRated, error) {
		return t.GetTVTopRated(options)
	}
)

// parseYearFromDate extracts the year from a TMDB date, returning 0 when the
// date is absent or malformed. It is meant for lightweight search/explorer
// results where a best-effort year is enough.
func parseYearFromDate(value string) int {
	if value == "" {
		return 0
	}
	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		return 0
	}
	return date.Year()
}

// mapMovieListItem builds a lightweight movie from a single TMDB list or
// search result. It never performs an additional TMDB request.
func mapMovieListItem(id int64, title, overview, releaseDate, poster, backdrop string, rating float32) *polochon.Movie {
	m := polochon.NewMovie(polochon.MovieConfig{})
	m.TmdbID = int(id)
	m.Title = title
	m.Plot = overview
	m.Year = parseYearFromDate(releaseDate)
	m.Rating = rating
	m.ThumbURL = imageURL(poster)
	m.FanartURL = imageURL(backdrop)
	return m
}

// mapShowListItem builds a lightweight show from a single TMDB list or search
// result. It never performs an additional TMDB request.
func mapShowListItem(id int64, name, overview, firstAirDate, poster, backdrop string, rating float32) *polochon.Show {
	s := polochon.NewShow(polochon.ShowConfig{})
	s.TmdbID = int(id)
	s.Title = name
	s.Plot = overview
	s.Year = parseYearFromDate(firstAirDate)
	s.Rating = rating
	s.PosterURL = imageURL(poster)
	s.FanartURL = imageURL(backdrop)
	s.BannerURL = s.FanartURL
	return s
}

// AvailableMovieOptions implements the Explorer interface
func (t *TmDB) AvailableMovieOptions() []string {
	return []string{
		movieOptionPopular,
		movieOptionTopRated,
		movieOptionUpcoming,
		movieOptionNowPlaying,
	}
}

// AvailableShowOptions implements the Explorer interface
func (t *TmDB) AvailableShowOptions() []string {
	return []string{
		showOptionPopular,
		showOptionTopRated,
		showOptionAiringToday,
		showOptionOnTheAir,
	}
}

// GetMovieList implements the Explorer interface. Each native category maps
// to exactly one TMDB list request; items stay lightweight and are hydrated
// only after selection.
func (t *TmDB) GetMovieList(_ context.Context, option string) ([]*polochon.Movie, error) {
	switch option {
	case movieOptionPopular:
		return t.listMoviePopular()
	case movieOptionTopRated:
		return t.listMovieTopRated()
	case movieOptionUpcoming:
		return t.listMovieUpcoming()
	case movieOptionNowPlaying:
		return t.listMovieNowPlaying()
	}
	return nil, ErrInvalidOption
}

// GetShowList implements the Explorer interface. Each native category maps to
// exactly one TMDB list request; items stay lightweight and are hydrated only
// after selection.
func (t *TmDB) GetShowList(_ context.Context, option string) ([]*polochon.Show, error) {
	switch option {
	case showOptionPopular:
		return t.listShowPopular()
	case showOptionTopRated:
		return t.listShowTopRated()
	case showOptionAiringToday:
		return t.listShowAiringToday()
	case showOptionOnTheAir:
		return t.listShowOnTheAir()
	}
	return nil, ErrInvalidOption
}

func (t *TmDB) listMoviePopular() ([]*polochon.Movie, error) {
	r, err := tmdbGetMoviePopular(t.client, map[string]string{})
	if err != nil {
		return nil, err
	}
	if r == nil || r.MoviePopularResults == nil {
		return []*polochon.Movie{}, nil
	}
	result := make([]*polochon.Movie, 0, len(r.Results))
	for _, item := range r.Results {
		result = append(result, mapMovieListItem(item.ID, item.Title, item.Overview, item.ReleaseDate, item.PosterPath, item.BackdropPath, item.VoteAverage))
	}
	return result, nil
}

func (t *TmDB) listMovieTopRated() ([]*polochon.Movie, error) {
	r, err := tmdbGetMovieTopRated(t.client, map[string]string{})
	if err != nil {
		return nil, err
	}
	if r == nil || r.MoviePopular == nil || r.MoviePopular.MoviePopularResults == nil {
		return []*polochon.Movie{}, nil
	}
	result := make([]*polochon.Movie, 0, len(r.Results))
	for _, item := range r.Results {
		result = append(result, mapMovieListItem(item.ID, item.Title, item.Overview, item.ReleaseDate, item.PosterPath, item.BackdropPath, item.VoteAverage))
	}
	return result, nil
}

func (t *TmDB) listMovieUpcoming() ([]*polochon.Movie, error) {
	r, err := tmdbGetMovieUpcoming(t.client, map[string]string{})
	if err != nil {
		return nil, err
	}
	if r == nil || r.MovieNowPlaying == nil || r.MovieNowPlaying.MovieNowPlayingResults == nil {
		return []*polochon.Movie{}, nil
	}
	result := make([]*polochon.Movie, 0, len(r.Results))
	for _, item := range r.Results {
		result = append(result, mapMovieListItem(item.ID, item.Title, item.Overview, item.ReleaseDate, item.PosterPath, item.BackdropPath, item.VoteAverage))
	}
	return result, nil
}

func (t *TmDB) listMovieNowPlaying() ([]*polochon.Movie, error) {
	r, err := tmdbGetMovieNowPlaying(t.client, map[string]string{})
	if err != nil {
		return nil, err
	}
	if r == nil || r.MovieNowPlayingResults == nil {
		return []*polochon.Movie{}, nil
	}
	result := make([]*polochon.Movie, 0, len(r.Results))
	for _, item := range r.Results {
		result = append(result, mapMovieListItem(item.ID, item.Title, item.Overview, item.ReleaseDate, item.PosterPath, item.BackdropPath, item.VoteAverage))
	}
	return result, nil
}

func (t *TmDB) listShowPopular() ([]*polochon.Show, error) {
	r, err := tmdbGetTVPopular(t.client, map[string]string{})
	if err != nil {
		return nil, err
	}
	if r == nil || r.TVAiringToday == nil || r.TVAiringToday.TVAiringTodayResults == nil {
		return []*polochon.Show{}, nil
	}
	result := make([]*polochon.Show, 0, len(r.Results))
	for _, item := range r.Results {
		result = append(result, mapShowListItem(item.ID, item.Name, item.Overview, item.FirstAirDate, item.PosterPath, item.BackdropPath, item.VoteAverage))
	}
	return result, nil
}

func (t *TmDB) listShowTopRated() ([]*polochon.Show, error) {
	r, err := tmdbGetTVTopRated(t.client, map[string]string{})
	if err != nil {
		return nil, err
	}
	if r == nil || r.TVAiringToday == nil || r.TVAiringToday.TVAiringTodayResults == nil {
		return []*polochon.Show{}, nil
	}
	result := make([]*polochon.Show, 0, len(r.Results))
	for _, item := range r.Results {
		result = append(result, mapShowListItem(item.ID, item.Name, item.Overview, item.FirstAirDate, item.PosterPath, item.BackdropPath, item.VoteAverage))
	}
	return result, nil
}

func (t *TmDB) listShowAiringToday() ([]*polochon.Show, error) {
	r, err := tmdbGetTVAiringToday(t.client, map[string]string{})
	if err != nil {
		return nil, err
	}
	if r == nil || r.TVAiringTodayResults == nil {
		return []*polochon.Show{}, nil
	}
	result := make([]*polochon.Show, 0, len(r.Results))
	for _, item := range r.Results {
		result = append(result, mapShowListItem(item.ID, item.Name, item.Overview, item.FirstAirDate, item.PosterPath, item.BackdropPath, item.VoteAverage))
	}
	return result, nil
}

func (t *TmDB) listShowOnTheAir() ([]*polochon.Show, error) {
	r, err := tmdbGetTVOnTheAir(t.client, map[string]string{})
	if err != nil {
		return nil, err
	}
	if r == nil || r.TVAiringToday == nil || r.TVAiringToday.TVAiringTodayResults == nil {
		return []*polochon.Show{}, nil
	}
	result := make([]*polochon.Show, 0, len(r.Results))
	for _, item := range r.Results {
		result = append(result, mapShowListItem(item.ID, item.Name, item.Overview, item.FirstAirDate, item.PosterPath, item.BackdropPath, item.VoteAverage))
	}
	return result, nil
}
