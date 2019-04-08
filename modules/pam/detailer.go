// Package pam - Polochon api module
package pam

import (
	"errors"

	"github.com/odwrtw/papi"
	"github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Module constants
const (
	moduleName = "pam"
)

// Pam module errors
var (
	ErrMissingArgument        = errors.New("pam: missing argument")
	ErrInvalidArgument        = errors.New("pam: invalid argument type")
	ErrMissingImdbID          = errors.New("pam: missing imdb id")
	ErrMissingEpisodeOrSeason = errors.New("pam: missing episode or season number")
	ErrInvalidType            = errors.New("pam: invalid type")
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

	client, err := papi.New(params.Endpoint)
	if err != nil {
		return nil, err
	}

	if params.Token != "" {
		client.SetToken(params.Token)
	}

	if params.BasicAuthUser != "" && params.BasicAuthPassword != "" {
		client.SetBasicAuth(params.BasicAuthUser, params.BasicAuthPassword)
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

// Status implements the Module interface
func (p *Pam) Status() (polochon.ModuleStatus, error) {
	return polochon.StatusNotImplemented, nil
}

// GetDetails implements the Detailer interface
func (p *Pam) GetDetails(i interface{}, log *logrus.Entry) error {
	switch resource := i.(type) {
	case *polochon.Movie:
		return p.getMovieDetails(resource)
	case *polochon.Show:
		return p.getShowDetails(resource)
	case *polochon.ShowEpisode:
		return p.getEpisodeDetails(resource)
	default:
		return ErrInvalidType
	}
}
