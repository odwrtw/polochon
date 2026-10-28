// Package simkl integrates Simkl movie and TV discovery with Polochon's existing modules.
package simkl

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-yaml"
	polochon "github.com/odwrtw/polochon/lib"
)

var (
	_ polochon.Explorer   = (*Simkl)(nil)
	_ polochon.Searcher   = (*Simkl)(nil)
	_ polochon.Detailer   = (*Simkl)(nil)
	_ polochon.Calendar   = (*Simkl)(nil)
	_ polochon.Wishlister = (*Simkl)(nil)

	ErrConnectionRequired = errors.New("simkl: connect an AUTH V2 account first")
	ErrScope              = errors.New("simkl: media:read scope required")
	ErrPlan               = errors.New("simkl: account plan does not support this operation")
	ErrQuota              = errors.New("simkl: upstream quota exhausted")
	ErrUnsupported        = errors.New("simkl: unsupported identity or operation")
	ErrAmbiguous          = errors.New("simkl: ambiguous identity")
	ErrNotFound           = errors.New("simkl: not found")
	ErrCredentials        = errors.New("simkl: authorization expired or revoked; run setup again")
)

func init() { polochon.RegisterModule(&Simkl{}) }

// Params uses the existing modules_params configuration mechanism.
type Params struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	RefreshToken string `yaml:"refresh_token"`
}

// Simkl shares its client and account snapshot across all configured roles.
type Simkl struct {
	mu            sync.Mutex
	client        *client
	log           *slog.Logger
	snapshotGate  chan struct{}
	snapshot      *wishlistSnapshot
	snapshotRetry time.Time
	snapshotErr   error
}

func (*Simkl) Name() string { return "simkl" }

func (s *Simkl) Init(raw []byte, log *slog.Logger) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != nil {
		return nil
	}
	var p Params
	if err := yaml.Unmarshal(raw, &p); err != nil {
		return err
	}
	if p.RefreshToken != "" && p.ClientID == "" {
		return errors.New("simkl: client_id required with refresh_token")
	}
	if log == nil {
		log = slog.Default()
	}
	s.log = log.With("module", "simkl")
	s.client = newClient(p)
	s.snapshotGate = make(chan struct{}, 1)
	return nil
}

func (s *Simkl) ready() (*client, error) {
	if err := s.Init(nil, nil); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client, nil
}

func (s *Simkl) Status() (polochon.ModuleStatus, error) {
	c, err := s.ready()
	if err != nil {
		return polochon.StatusFail, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var items []record
	err = c.get(ctx, c.cdn+"/discover/trending/movies/today_100.json", false, time.Hour, true, &items)
	if err != nil {
		return polochon.StatusFail, err
	}
	return polochon.StatusOK, nil
}

func appVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
		for _, v := range info.Settings {
			if v.Key == "vcs.revision" && len(v.Value) >= 12 {
				return v.Value[:12]
			}
		}
	}
	return "dev"
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		// API calls never need to follow redirects to a website or artwork host.
		return http.ErrUseLastResponse
	}}
}

func hasScope(scope string) bool {
	for _, v := range strings.Fields(scope) {
		if v == "media:read" {
			return true
		}
	}
	return false
}
