package trakttv

import (
	"errors"

	"gopkg.in/yaml.v2"

	"github.com/Sirupsen/logrus"
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
	polochon.RegisterDetailer(moduleName, NewFromRawYaml)
}

// Params represents the module params
type Params struct {
	ClientID string `yaml:"client_id"`
}

// TraktTV implements the detailer interface
type TraktTV struct {
	client *trakttv.TraktTv
}

// NewFromRawYaml creates a new TraktTV from YML config
func NewFromRawYaml(p []byte) (polochon.Detailer, error) {
	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return nil, err
	}

	return New(params)
}

// New returns a new TraktTV
func New(p *Params) (*TraktTV, error) {
	return &TraktTV{
		client: trakttv.New(p.ClientID),
	}, nil
}

// Name returns the module name
func (trakt *TraktTV) Name() string {
	return moduleName
}

// GetDetails gets details for the polochon video object
func (trakt *TraktTV) GetDetails(i interface{}, log *logrus.Entry) error {
	movie, ok := i.(*polochon.Movie)
	if !ok {
		return ErrInvalidArgument
	}

	tmovie, err := trakt.client.SearchMovieByID(movie.ImdbID, trakttv.QueryOption{
		ExtendedInfos: []trakttv.ExtendedInfo{
			trakttv.ExtendedInfoFull,
			trakttv.ExtendedInfoImages,
		},
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
	movie.Thumb = tmovie.Images.Poster.Full
	movie.Fanart = tmovie.Images.Fanart.Full

	return nil
}
