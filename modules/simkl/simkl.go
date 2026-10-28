package simkl

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/goccy/go-yaml"

	polochon "github.com/odwrtw/polochon/lib"
)

const moduleName = "simkl"

var (
	ErrMissingClientID = errors.New("simkl: missing client_id")
	ErrInvalidArgument = errors.New("simkl: invalid argument")
	ErrInvalidResponse = errors.New("simkl: invalid response")
	ErrNotFound        = errors.New("simkl: not found")
	ErrMissingArtwork  = errors.New("simkl: missing required artwork")
)

// Params identifies the application, not a Simkl user. No user authentication is used.
type Params struct {
	ClientID   string `yaml:"client_id"`
	AppVersion string `yaml:"app_version"`
}

// Simkl provides public movie and TV metadata. The separate anime catalog is excluded.
type Simkl struct {
	client *apiClient
}

func init() { polochon.RegisterModule(&Simkl{}) }

func (s *Simkl) Name() string { return moduleName }

func (s *Simkl) Init(data []byte, _ *slog.Logger) error {
	if s.client != nil {
		return nil
	}
	params := &Params{}
	if err := yaml.Unmarshal(data, params); err != nil {
		return err
	}
	return s.InitWithParams(params)
}

func (s *Simkl) InitWithParams(params *Params) error {
	if s.client != nil {
		return nil
	}
	if params == nil || strings.TrimSpace(params.ClientID) == "" {
		return ErrMissingClientID
	}
	version := strings.TrimSpace(params.AppVersion)
	if strings.ContainsAny(version, "\r\n") {
		return ErrInvalidArgument
	}
	if version == "" {
		version = "dev"
	}
	s.client = newAPIClient(strings.TrimSpace(params.ClientID), version)
	return nil
}

func (s *Simkl) Status() (polochon.ModuleStatus, error) {
	if s.client == nil {
		return polochon.StatusFail, ErrMissingClientID
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Search is uncached and validates the client ID, unlike the public CDN feeds.
	items, err := s.client.search(ctx, "movie", "The Matrix")
	if err == nil && len(items) == 0 {
		err = ErrNotFound
	}
	if err != nil {
		return polochon.StatusFail, err
	}
	return polochon.StatusOK, nil
}
