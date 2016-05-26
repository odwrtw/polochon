// Package pam - Polochon api module
package pam

import (
	"errors"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/papi"
	"github.com/odwrtw/polochon/lib"
	"gopkg.in/yaml.v2"
)

// Module constants
const (
	moduleName = "pam"
)

// Pam module errors
var (
	ErrMissingArgument = errors.New("pam: missing argument")
	ErrInvalidArgument = errors.New("pam: invalid argument type")
	ErrMissingImdbID   = errors.New("pam: missing imdb id")
)

// Params represents the module params
type Params struct {
	Endpoint          string `yaml:"endpoint"`
	Token             string `yaml:"token"`
	BasicAuthUser     string `yaml:"basic_auth_user"`
	BasicAuthPassword string `yaml:"basic_auth_password"`
}

// Register a new notifier
func init() {
	polochon.RegisterDetailer(moduleName, NewDetailerFromRawYaml)
}

// NewFromRawYaml unmarshals the bytes as yaml as params and call the New
// function
func NewFromRawYaml(p []byte) (*Pam, error) {
	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return nil, err
	}

	return New(params)
}

// NewDetailerFromRawYaml returns a new detailer from raw yaml config
func NewDetailerFromRawYaml(p []byte) (polochon.Detailer, error) {
	return NewDetailerFromRawYaml(p)
}

// New returns a new pam
func New(params *Params) (*Pam, error) {
	if params.Endpoint == "" {
		return nil, ErrMissingArgument
	}

	if params.Token == "" {
		return nil, ErrMissingArgument
	}

	client := &papi.Client{}
	var err error
	if params.BasicAuthUser != "" && params.BasicAuthPassword != "" {
		client, err = papi.NewWithBasicAuth(params.Endpoint, params.Token, params.BasicAuthUser, params.BasicAuthPassword)
	} else {
		client, err = papi.New(params.Endpoint, params.Token)
	}

	if err != nil {
		return nil, err
	}

	return &Pam{client: client}, nil
}

// NewDetailer returns a new detailer
func NewDetailer(params *Params) (polochon.Detailer, error) {
	return New(params)
}

// Pam represents a detailer based on the polochon informations
type Pam struct {
	client *papi.Client
}

// Name implements the Module interface
func (p *Pam) Name() string {
	return moduleName
}

// GetDetails implements the Detailer interface
func (p *Pam) GetDetails(i interface{}, log *logrus.Entry) error {
	m, ok := i.(*polochon.Movie)
	if !ok {
		return ErrInvalidArgument
	}

	if m.ImdbID == "" {
		return ErrMissingImdbID
	}

	movie, err := p.client.MovieByID(m.ImdbID)
	if err != nil {
		return err
	}

	m.OriginalTitle = movie.OriginalTitle
	m.Plot = movie.Plot
	m.Rating = movie.Rating
	m.Runtime = movie.Runtime
	m.SortTitle = movie.SortTitle
	m.Tagline = movie.Tagline
	m.Thumb = movie.Thumb
	m.Fanart = movie.Fanart
	m.Title = movie.Title
	m.TmdbID = movie.TmdbID
	m.Votes = movie.Votes
	m.Year = movie.Year

	return nil
}
