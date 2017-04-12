package trakttv

import (
	"errors"
	"strconv"

	"gopkg.in/yaml.v2"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/fanarttv"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/trakttv"
)

const (
	moduleName = "trakttv"
)

var (
	// ErrInvalidArgument returned when an invalid object is passed to GetDetails
	ErrInvalidArgument = errors.New("trakttv: invalid argument")
)

// Register trakttv as a Detailer
func init() {
	polochon.RegisterDetailer(moduleName, NewDetailer)
}

// Params represents the module params
type Params struct {
	ClientID       string `yaml:"client_id"`
	FanartTvAPIKey string `yaml:"fanarttv_api_key"`
}

// TraktTV implements the detailer interface
type TraktTV struct {
	client       *trakttv.TraktTv
	fanartClient *fanarttv.Client
}

// NewDetailer creates a new TraktTV Detailer
func NewDetailer(p []byte) (polochon.Detailer, error) {
	return NewFromRawYaml(p)
}

// NewFromRawYaml creates a new TraktTV from YML config
func NewFromRawYaml(p []byte) (*TraktTV, error) {
	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return nil, err
	}

	return New(params)
}

// New returns a new TraktTV
func New(p *Params) (*TraktTV, error) {
	return &TraktTV{
		client:       trakttv.New(p.ClientID),
		fanartClient: fanarttv.New(p.FanartTvAPIKey),
	}, nil
}

// Name returns the module name
func (trakt *TraktTV) Name() string {
	return moduleName
}

// GetDetails gets details for the polochon video object
func (trakt *TraktTV) GetDetails(i interface{}, log *logrus.Entry) error {
	var err error
	switch v := i.(type) {
	case *polochon.Show:
		err = trakt.getShowDetails(v, log)
	case *polochon.Movie:
		err = trakt.getMovieDetails(v, log)
	default:
		return ErrInvalidArgument
	}

	return err
}

// getMovieDetails gets details for the polochon movie object
func (trakt *TraktTV) getMovieDetails(movie *polochon.Movie, log *logrus.Entry) error {
	tmovie, err := trakt.client.SearchMovieByID(movie.ImdbID, trakttv.QueryOption{
		ExtendedInfos: []trakttv.ExtendedInfo{trakttv.ExtendedInfoFull},
	})
	if err != nil {
		return err
	}

	// Update movie details
	movie.TmdbID = tmovie.IDs.TmDB
	movie.OriginalTitle = tmovie.Title
	movie.SortTitle = tmovie.Title
	movie.Title = tmovie.Title
	movie.Plot = tmovie.Overview
	movie.Tagline = tmovie.Tagline
	movie.Votes = tmovie.Votes
	movie.Rating = float32(tmovie.Rating)
	movie.Year = tmovie.Year
	movie.Genres = tmovie.Genres

	// Search for images
	res, err := trakt.fanartClient.GetMovieImages(movie.ImdbID)
	if err != nil {
		return err
	}

	thumb := fanarttv.Best(res.Posters)
	if thumb != nil {
		movie.Thumb = thumb.URL
	}

	fanart := fanarttv.Best(res.Backgrounds)
	if fanart != nil {
		movie.Fanart = fanart.URL
	}

	return nil
}

// getShowDetails gets details for the polochon show object
func (trakt *TraktTV) getShowDetails(show *polochon.Show, log *logrus.Entry) error {
	tshow, err := trakt.client.SearchShowByID(show.ImdbID, trakttv.QueryOption{
		ExtendedInfos: []trakttv.ExtendedInfo{trakttv.ExtendedInfoFull},
	})
	if err != nil {
		return err
	}

	// Update show details
	show.TvdbID = tshow.IDs.TvDB
	show.Title = tshow.Title
	show.Year = tshow.Year
	show.Plot = tshow.Overview
	show.FirstAired = &tshow.FirstAired
	show.Rating = float32(tshow.Rating)

	// Search for images
	res, err := trakt.fanartClient.GetShowImages(strconv.Itoa(tshow.IDs.TvDB))
	if err != nil {
		return err
	}

	fanart := fanarttv.Best(res.Backgrounds)
	if fanart != nil {
		show.Fanart = fanart.URL
	}

	poster := fanarttv.Best(res.Posters)
	if poster != nil {
		show.Poster = poster.URL
	}

	banner := fanarttv.Best(res.Banners)
	if banner != nil {
		show.Banner = banner.URL
	}

	return nil
}
