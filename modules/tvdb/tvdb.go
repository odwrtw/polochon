package tvdb

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/pioz/tvdb"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Register tvdb as a Detailer
func init() {
	polochon.RegisterDetailer(moduleName, NewDetailerFromRawYaml)
	polochon.RegisterCalendar(moduleName, NewCalendarFromRawYaml)
}

// Module constants
const (
	moduleName      = "tvdb"
	tokenExpiration = 24 * time.Hour
)

// Errors
var (
	ErrShowNotFound                   = errors.New("tvdb: show not found")
	ErrShowImageNotFound              = errors.New("tvdb: show image not found")
	ErrNotEnoughArguments             = errors.New("tvdb: not enough arguments to perform search")
	ErrInvalidArgument                = errors.New("tvdb: invalid argument type")
	ErrMissingShowEpisodeInformations = errors.New("tvdb: missing show episode informations to get details")
	ErrFailedToUpdateEpisode          = errors.New("tvdb: failed to update episode details")
)

// Params represents the module params
type Params struct {
	ApiKey   string `yaml:"api_key"`
	UserID   string `yaml:"user_id"`
	Username string `yaml:"username"`
}

// TvDB implents the Detailer interface
type TvDB struct {
	client           *tvdb.Client
	lastTokenRefresh *time.Time
}

// NewFromRawYaml creates a new TvDB from YML config
func NewFromRawYaml(p []byte) (*TvDB, error) {
	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return nil, err
	}

	return New(params)
}

// New returns a new tvdb module
func New(params *Params) (*TvDB, error) {
	return &TvDB{
		client: &tvdb.Client{
			Apikey:   params.ApiKey,
			Username: params.Username,
			Userkey:  params.UserID,
		},
	}, nil
}

// NewDetailerFromRawYaml implements the detailer interface
func NewDetailerFromRawYaml(p []byte) (polochon.Detailer, error) {
	return NewFromRawYaml(p)
}

// NewCalendarFromRawYaml implements the calendar interface
func NewCalendarFromRawYaml(p []byte) (polochon.Calendar, error) {
	return NewFromRawYaml(p)
}

// Name implements the Module interface
func (t *TvDB) Name() string {
	return moduleName
}

// login handles the token refresh
func (t *TvDB) login() error {
	if t.lastTokenRefresh != nil && time.Since(*t.lastTokenRefresh) < tokenExpiration/2 {
		// The token is still valid
		return nil
	}

	// Never logged in or the token has expired
	var f func() error
	if t.lastTokenRefresh == nil || time.Since(*t.lastTokenRefresh) > tokenExpiration {
		f = t.client.Login
	} else {
		f = t.client.RefreshToken
	}

	if err := f(); err != nil {
		t.lastTokenRefresh = nil
		return err
	}

	now := time.Now()
	t.lastTokenRefresh = &now
	return nil
}

// Status implements the Module interface
func (t *TvDB) Status() (polochon.ModuleStatus, error) {
	// Search for The Matrix on trakttv via imdbID
	_, err := t.searchByImdbID("tt2085059")
	if err != nil {
		return polochon.StatusFail, err
	}

	return polochon.StatusOK, nil
}

func (t *TvDB) searchByImdbID(id string) (*tvdb.Series, error) {
	if err := t.login(); err != nil {
		return nil, err
	}

	series, err := t.client.SearchByImdbID(id)
	if err != nil {
		if tvdb.HaveCodeError(404, err) {
			return nil, ErrShowNotFound
		}

		return nil, err
	}

	if len(series) == 0 {
		return nil, ErrShowNotFound
	}

	return &series[0], nil
}

func (t *TvDB) searchByName(query string) (*tvdb.Series, error) {
	if err := t.login(); err != nil {
		return nil, err
	}

	series, err := t.client.BestSearch(query)
	if err != nil {
		if tvdb.HaveCodeError(404, err) {
			return nil, ErrShowNotFound
		}

		return nil, err
	}

	return &series, nil
}

// GetDetails implements the Detailer interface
func (t *TvDB) GetDetails(i interface{}, log *logrus.Entry) error {
	switch v := i.(type) {
	case *polochon.Show:
		return t.getShowDetails(v, nil)
	case *polochon.ShowEpisode:
		return t.getEpisodeDetails(v)
	default:
		return ErrInvalidArgument
	}
}

func (t *TvDB) searchShow(s *polochon.Show) (*tvdb.Series, error) {
	var show *tvdb.Series
	var err error

	switch {
	case s.ImdbID != "":
		show, err = t.searchByImdbID(s.ImdbID)
	case s.Title != "":
		// Add the year to the search if defined
		query := s.Title
		if s.Year != 0 {
			query = fmt.Sprintf("%s (%d)", query, s.Year)
		}

		show, err = t.searchByName(query)
	default:
		return nil, ErrNotEnoughArguments
	}

	if err != nil {
		return nil, err
	}

	if show == nil {
		return nil, ErrShowNotFound
	}

	return show, nil
}

func (t *TvDB) getShowEpisodes(s *polochon.Show, show *tvdb.Series, params url.Values) error {
	err := t.client.GetSeriesEpisodes(show, params)
	if err != nil {
		return err
	}

	// Get runtime
	var runtime int
	if show.Runtime != "" {
		r, err := strconv.Atoi(show.Runtime)
		if err != nil {
			return err
		}
		runtime = r
	}

	// Go through each episode from the list
	s.Episodes = []*polochon.ShowEpisode{}
	for _, e := range show.Episodes {
		episode := polochon.NewShowEpisode(polochon.ShowConfig{})
		episode.Title = e.EpisodeName
		episode.ShowTitle = s.Title
		episode.Season = e.AiredSeason
		episode.Episode = e.AiredEpisodeNumber
		episode.TvdbID = e.ID
		episode.Aired = e.FirstAired
		episode.Plot = e.Overview
		episode.Runtime = runtime
		episode.Thumb = tvdb.ImageURL(e.Filename)
		episode.ShowImdbID = s.ImdbID
		episode.ShowTvdbID = s.TvdbID
		episode.EpisodeImdbID = e.ImdbID
		episode.Rating = e.SiteRating

		// Add the episode to the list
		s.Episodes = append(s.Episodes, episode)
	}

	return nil
}

func (t *TvDB) getShowImages(s *polochon.Show, show *tvdb.Series) error {
	// Update the banner
	s.Banner = show.BannerURL()

	// Update the poster and fannart
	for _, imageType := range []struct {
		f   func(s *tvdb.Series) error
		url *string
		t   string
	}{
		{
			t:   "fanart",
			url: &s.Fanart,
			f:   t.client.GetSeriesFanartImages,
		},
		{
			t:   "poster",
			url: &s.Poster,
			f:   t.client.GetSeriesPosterImages,
		},
	} {
		// Fetch the image
		if err := imageType.f(show); err != nil {
			return err
		}

		images := []*tvdb.Image{}
		for _, i := range show.Images {
			if i.KeyType != imageType.t {
				continue
			}
			images = append(images, &i)
		}

		if len(images) == 0 {
			return ErrShowImageNotFound
		}

		// Sort images by ratings and count
		sort.Slice(images, func(i, j int) bool {
			avgI := images[i].RatingsInfo.Average
			avgJ := images[j].RatingsInfo.Average
			if avgI == avgJ {
				countI := images[i].RatingsInfo.Count
				countJ := images[j].RatingsInfo.Count
				return countI > countJ
			}
			return avgI > avgJ
		})

		// Update the image url
		*imageType.url = tvdb.ImageURL(images[0].FileName)
	}

	return nil
}

func (t *TvDB) getShowDetails(s *polochon.Show, params url.Values) error {
	// Search for the show
	show, err := t.searchShow(s)
	if err != nil {
		return err
	}

	// Get the show details
	err = t.client.GetSeries(show)
	if err != nil {
		return err
	}

	s.Title = show.SeriesName
	s.Plot = show.Overview
	s.TvdbID = show.ID
	s.ImdbID = show.ImdbID
	s.Rating = show.SiteRating

	// Get the year from the first aired date
	if show.FirstAired != "" {
		date, err := time.Parse("2006-01-02", show.FirstAired)
		if err != nil {
			return err
		}
		s.Year = date.Year()
		s.FirstAired = &date
	}

	// Get show images
	if err := t.getShowImages(s, show); err != nil {
		return err
	}

	return t.getShowEpisodes(s, show, params)
}

func (t *TvDB) getEpisodeDetails(s *polochon.ShowEpisode) error {
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
	// Copy missing informations
	if show.Title == "" && s.ShowTitle != "" {
		show.Title = s.ShowTitle
	}
	if show.ImdbID == "" && s.ShowImdbID != "" {
		show.ImdbID = s.ShowImdbID
	}

	params := url.Values{
		"airedSeason":  {strconv.Itoa(s.Season)},
		"airedEpisode": {strconv.Itoa(s.Episode)},
	}

	err := t.getShowDetails(show, params)
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

// GetShowCalendar implements the Calendar interface
func (t *TvDB) GetShowCalendar(show *polochon.Show, log *logrus.Entry) (*polochon.ShowCalendar, error) {
	if err := t.getShowDetails(show, nil); err != nil {
		return nil, err
	}

	if show == nil || show.ImdbID == "" {
		return nil, ErrShowNotFound
	}
	calendar := polochon.NewShowCalendar(show.ImdbID)

	// Get show details
	for _, e := range show.Episodes {
		var aired time.Time
		var err error
		if e.Aired != "" {
			aired, err = time.Parse("2006-01-02", e.Aired)
			if err != nil {
				return nil, err
			}
		}

		calendar.Episodes = append(calendar.Episodes, &polochon.ShowCalendarEpisode{
			Season:    e.Season,
			Episode:   e.Episode,
			AiredDate: &aired,
		})
	}

	return calendar, nil
}
