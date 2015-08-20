package tvdb

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/arbovm/levenshtein"
	"github.com/garfunkel/go-tvdb"
	"github.com/odwrtw/polochon/lib"
)

// Register tvdb as a Detailer
func init() {
	polochon.RegisterDetailer(moduleName, NewDetailer)
	polochon.RegisterCalendar(moduleName, NewCalendar)
}

// API constants
const (
	APIendpoint = "http://www.thetvdb.com/api"
	Token       = "1D62F2F90030C444"
	AssetsURL   = "http://thetvdb.com/banners/"
)

// Module constants
const (
	moduleName = "tvdb"
)

// Errors
var (
	ErrShowNotFound                   = errors.New("tvdb: show not found")
	ErrSeasonNotFound                 = errors.New("tvdb: season not found")
	ErrEpisodeNotFound                = errors.New("tvdb: episode not found")
	ErrShowUpdate                     = errors.New("tvdb: failed to get details for show")
	ErrMissingShowTitle               = errors.New("tvdb: missing show title")
	ErrMissingShowImdbID              = errors.New("tvdb: missing show imdb id")
	ErrNotEnoughArguments             = errors.New("tvdb: not enough arguments to perform search")
	ErrInvalidArgument                = errors.New("tvdb: invalid argument type")
	ErrMissingShowEpisodeInformations = errors.New("tvdb: missing show episode informations to get details")
	ErrFailedToUpdateEpisode          = errors.New("tvdb: failed to update episode details")
)

// TvDB implents the Detailer interface
type TvDB struct {
	log *logrus.Entry
}

// NewDetailer returns an initialized tmdb instance as a detailer
func NewDetailer(params map[string]interface{}, log *logrus.Entry) (polochon.Detailer, error) {
	return &TvDB{log: log}, nil
}

// NewCalendar returns an initialized tmdb instance as a calendar
func NewCalendar(params map[string]interface{}, log *logrus.Entry) (polochon.Calendar, error) {
	return &TvDB{log: log}, nil
}

// assetURL return the URL of an asset
func assetURL(asset string) string {
	return AssetsURL + asset
}

// Get show detail
var getShowFromTvdb = func(s *tvdb.Series) error {
	return s.GetDetail()
}

// updateShow update the show information from the tvdb infos
func updateShow(s *polochon.Show, tvSeries *tvdb.Series) error {
	err := getShowFromTvdb(tvSeries)
	if err != nil {
		return err
	}

	s.Title = tvSeries.SeriesName
	s.ShowTitle = tvSeries.SeriesName
	s.Plot = tvSeries.Overview
	s.TvdbID = int(tvSeries.ID)
	s.URL = fmt.Sprintf("%s/%s/series/%d/all/en.zip", APIendpoint, Token, s.TvdbID)
	s.ImdbID = tvSeries.ImdbID
	s.Banner = assetURL(tvSeries.Banner)
	s.Fanart = assetURL(tvSeries.Fanart)
	s.Poster = assetURL(tvSeries.Poster)

	// Get the year from the first aired date
	if tvSeries.FirstAired != "" {
		date, err := time.Parse("2006-01-02", tvSeries.FirstAired)
		if err != nil {
			return err
		}
		s.Year = date.Year()
	}

	// Convert rating string to a float
	if tvSeries.Rating != "" {
		f, err := strconv.ParseFloat(tvSeries.Rating, 32)
		if err != nil {
			return err
		}
		s.Rating = float32(f)
	}

	// Get runtime
	var runtime int
	if tvSeries.Runtime != "" {
		r, err := strconv.Atoi(tvSeries.Runtime)
		if err != nil {
			return err
		}
		runtime = r
	}

	// Go through each episode from the list
	s.Episodes = []*polochon.ShowEpisode{}
	for _, episodeList := range tvSeries.Seasons {
		for _, e := range episodeList {
			episode := polochon.NewShowEpisode(polochon.ShowConfig{})
			episode.Title = e.EpisodeName
			episode.ShowTitle = s.Title
			episode.Season = int(e.SeasonNumber)
			episode.Episode = int(e.EpisodeNumber)
			episode.TvdbID = int(e.ID)
			episode.Aired = e.FirstAired
			episode.Plot = e.Overview
			episode.Runtime = runtime
			episode.Thumb = assetURL(e.Filename)
			episode.ShowImdbID = s.ImdbID
			episode.ShowTvdbID = s.TvdbID
			episode.EpisodeImdbID = e.ImdbID

			if e.Rating != "" {
				f, err := strconv.ParseFloat(e.Rating, 32)
				if err != nil {
					return err
				}
				episode.Rating = float32(f)
			}

			// Add the episode to the list
			s.Episodes = append(s.Episodes, episode)
		}
	}

	return nil
}

// Function to be overwritten during the tests
var tvdbGetSeries = func(name string) (seriesList tvdb.SeriesList, err error) {
	return tvdb.GetSeries(name)
}

// getShowByName helps find a show on tvdb using its name
func getShowByName(s *polochon.Show) error {
	if s.Title == "" {
		return ErrMissingShowTitle
	}

	// Add the year to the search if defined
	query := s.Title
	if s.Year != 0 {
		query = fmt.Sprintf("%s (%d)", query, s.Year)
	}

	// Search on tvdb by name
	list, err := tvdbGetSeries(query)
	if err != nil {
		return err
	}

	// Any result ?
	if len(list.Series) == 0 {
		return ErrShowNotFound
	}

	// Find the most accurate serie base on the levenshtein distance
	var show *tvdb.Series
	minDistance := 100
	for _, tvdbSerie := range list.Series {
		d := levenshtein.Distance(query, tvdbSerie.SeriesName)
		if d < minDistance {
			minDistance = d
			show = tvdbSerie
		}
	}

	return updateShow(s, show)
}

// Function to be overwritten during the tests
var tvdbGetShowByImdbID = func(id string) (series *tvdb.Series, err error) {
	return tvdb.GetSeriesByIMDBID(id)
}

// get show by imdb id
func getShowByImdbID(s *polochon.Show) error {
	if s.ImdbID == "" {
		return ErrMissingShowImdbID
	}

	show, err := tvdbGetShowByImdbID(s.ImdbID)
	if err != nil {
		return err
	}

	return updateShow(s, show)
}

// getShowDetails
func getShowDetails(s *polochon.Show) error {
	switch {
	case s.ImdbID != "":
		return getShowByImdbID(s)
	case s.Title != "":
		return getShowByName(s)
	default:
		return ErrNotEnoughArguments
	}
}

// getShowDetails
func getShowEpisodeDetails(s *polochon.ShowEpisode) error {
	// The season / episode infos are needed
	if s.Season == 0 || s.Episode == 0 {
		return ErrMissingShowEpisodeInformations
	}

	// The show should be found by title or imdb id
	if s.ShowTitle == "" && s.ShowImdbID == "" {
		return ErrMissingShowEpisodeInformations
	}

	// Use the show included in the episode if present
	var show *polochon.Show
	if s.Show != nil {
		show = s.Show
	} else {
		show = polochon.NewShow(polochon.ShowConfig{})
	}
	show.Title = s.ShowTitle
	show.ImdbID = s.ShowImdbID

	err := getShowDetails(show)
	if err != nil {
		return err
	}

	var updated bool
	for _, e := range show.Episodes {
		if e.Season == s.Season && e.Episode == s.Episode {
			s.Title = e.Title
			s.ShowTitle = e.ShowTitle
			s.Season = e.Season
			s.Episode = e.Episode
			s.TvdbID = e.TvdbID
			s.Aired = e.Aired
			s.Plot = e.Plot
			s.Runtime = e.Runtime
			s.Thumb = e.Thumb
			s.Rating = e.Rating
			s.ShowImdbID = e.ShowImdbID
			s.ShowTvdbID = e.ShowTvdbID
			s.EpisodeImdbID = e.EpisodeImdbID

			updated = true
			break
		}
	}

	if !updated {
		return ErrFailedToUpdateEpisode
	}

	return nil
}

// Name implements the Module interface
func (t *TvDB) Name() string {
	return moduleName
}

// GetDetails implements the Detailer interface
func (t *TvDB) GetDetails(i interface{}) error {
	switch v := i.(type) {
	case *polochon.Show:
		return getShowDetails(v)
	case *polochon.ShowEpisode:
		return getShowEpisodeDetails(v)
	default:
		return ErrInvalidArgument
	}
}

// GetShowCalendar implements the Calendar interface
func (t *TvDB) GetShowCalendar(show *polochon.Show) (*polochon.ShowCalendar, error) {
	// Get show details
	if show.ImdbID == "" {
		return nil, ErrMissingShowImdbID
	}
	calendar := polochon.NewShowCalendar(show.ImdbID)

	s, err := tvdb.GetSeriesByIMDBID(show.ImdbID)
	if err != nil {
		return nil, err
	}

	if err := s.GetDetail(); err != nil {
		return nil, err
	}

	for s, es := range s.Seasons {
		for _, e := range es {
			episodeCalendar := &polochon.ShowCalendarEpisode{
				Season:  int(s),
				Episode: int(e.EpisodeNumber),
			}

			// Parse aired date
			if e.FirstAired != "" {
				aired, err := time.Parse("2006-01-02", e.FirstAired)
				if err != nil {
					return nil, err
				}
				episodeCalendar.AiredDate = &aired
			}

			calendar.Episodes = append(calendar.Episodes, episodeCalendar)
		}
	}

	return calendar, nil
}
