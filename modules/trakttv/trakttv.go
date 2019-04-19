package trakttv

import (
	"errors"

	"gopkg.in/yaml.v2"

	"github.com/odwrtw/fanarttv"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/trakttv"
)

// Make sure that the module is a detailer, an explorer and a searcher
var (
	_ polochon.Detailer = (*TraktTV)(nil)
	_ polochon.Explorer = (*TraktTV)(nil)
	_ polochon.Searcher = (*TraktTV)(nil)
)

// Register trakttv as a Detailer
func init() {
	polochon.RegisterModule(&TraktTV{})
}

const (
	moduleName = "trakttv"
)

var (
	// ErrInvalidArgument returned when an invalid object is passed to GetDetails
	ErrInvalidArgument = errors.New("trakttv: invalid argument")
	// ErrNotFound returned when a search returns no results
	ErrNotFound = errors.New("trakttv: not found")
)

// Params represents the module params
type Params struct {
	ClientID       string `yaml:"client_id"`
	FanartTvAPIKey string `yaml:"fanarttv_api_key"`
}

// TraktTV implements the detailer interface
type TraktTV struct {
	client       *trakttv.TraktTv
	fanartClient *fanarttv.Client
	configured   bool
}

// Init implements the module interface
func (trakt *TraktTV) Init(p []byte) error {
	if trakt.configured {
		return nil
	}

	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return err
	}

	return trakt.InitWithParams(params)
}

// InitWithParams configures the module
func (trakt *TraktTV) InitWithParams(params *Params) error {
	trakt.client = trakttv.New(params.ClientID)
	trakt.fanartClient = fanarttv.New(params.FanartTvAPIKey)
	trakt.configured = true
	return nil
}

// Name returns the module name
func (trakt *TraktTV) Name() string {
	return moduleName
}

// Status implements the Module interface
func (trakt *TraktTV) Status() (polochon.ModuleStatus, error) {
	status := polochon.StatusOK

	// Search for The Matrix on trakttv via imdbID
	_, err := trakt.client.SearchMovieByID("tt0133093", trakttv.QueryOption{})
	if err != nil {
		status = polochon.StatusFail
	}
	return status, err
}
