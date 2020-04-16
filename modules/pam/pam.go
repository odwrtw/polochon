package pam

import (
	"errors"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/papi"
	"gopkg.in/yaml.v2"
)

// Make sure that the module is a detailer
var _ polochon.Detailer = (*Pam)(nil)

func init() {
	polochon.RegisterModule(&Pam{})
}

// Module constants
const (
	moduleName = "pam"
)

// Pam module errors
var (
	ErrMissingArgument        = errors.New("pam: missing argument")
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

// Pam represents a detailer based on the polochon informations
type Pam struct {
	client     *papi.Client
	configured bool
}

// Init implements the module interface
func (p *Pam) Init(data []byte) error {
	if p.configured {
		return nil
	}

	params := &Params{}
	if err := yaml.Unmarshal(data, params); err != nil {
		return err
	}

	return p.InitWithParams(params)
}

// InitWithParams configures the module
func (p *Pam) InitWithParams(params *Params) error {
	if params.Endpoint == "" {
		return ErrMissingArgument
	}

	client, err := papi.New(params.Endpoint)
	if err != nil {
		return err
	}

	if params.Token != "" {
		client.SetToken(params.Token)
	}

	if params.BasicAuthUser != "" && params.BasicAuthPassword != "" {
		client.SetBasicAuth(params.BasicAuthUser, params.BasicAuthPassword)
	}

	p.client = client
	p.configured = true

	return nil
}

// Name implements the Module interface
func (p *Pam) Name() string {
	return moduleName
}

// Status implements the Module interface
func (p *Pam) Status() (polochon.ModuleStatus, error) {
	return polochon.StatusNotImplemented, nil
}
