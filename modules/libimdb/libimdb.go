// Package libimdb assists detailing and searches an externally maintained IMDb catalog.
package libimdb

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	imdb "github.com/odwrtw/libimdb"

	polochon "github.com/odwrtw/polochon/lib"
)

const (
	moduleName   = "libimdb"
	queryTimeout = 5 * time.Second
)

var (
	ErrMissingDBPath   = errors.New("libimdb: missing database path")
	ErrInvalidDBPath   = errors.New("libimdb: invalid database path")
	ErrNotConfigured   = errors.New("libimdb: module not configured")
	ErrInvalidArgument = errors.New("libimdb: invalid argument")
	ErrKindMismatch    = errors.New("libimdb: IMDb ID has a different title kind")
	ErrConflictingIDs  = errors.New("libimdb: conflicting parent IMDb IDs")

	_ polochon.ContinuingDetailer = (*Catalog)(nil)
	_ polochon.Searcher           = (*Catalog)(nil)
)

// Params configures a catalog created and refreshed outside Polochon.
type Params struct {
	DBPath string `yaml:"db_path"`
}

// Catalog supplies matching assistance, not complete provider enrichment.
type Catalog struct {
	dbPath     string
	log        *slog.Logger
	configured bool
}

func init() {
	polochon.RegisterModule(&Catalog{})
}

func (c *Catalog) Name() string { return moduleName }

func (c *Catalog) Init(p []byte, log *slog.Logger) error {
	if c.configured {
		return nil
	}
	if log == nil {
		log = slog.Default()
	}
	c.log = log.With("module", moduleName)
	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return err
	}
	return c.InitWithParams(params)
}

func (c *Catalog) InitWithParams(params *Params) error {
	if c.configured {
		return nil
	}
	if params == nil || params.DBPath == "" {
		return ErrMissingDBPath
	}
	if strings.ContainsRune(params.DBPath, 0) {
		return ErrInvalidDBPath
	}
	if c.log == nil {
		c.log = slog.Default().With("module", moduleName)
	}
	// Availability is checked at query time, so missing catalogs do not turn
	// optional matching assistance into a startup prerequisite.
	c.dbPath = params.DBPath
	c.configured = true
	return nil
}

func (c *Catalog) ContinueOnSuccess() bool { return true }

func (c *Catalog) open(ctx context.Context) (*imdb.Catalog, error) {
	if !c.configured {
		return nil, ErrNotConfigured
	}
	return imdb.Open(ctx, c.dbPath)
}

func (c *Catalog) Status() (polochon.ModuleStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()
	catalog, err := c.open(ctx)
	if err != nil {
		return polochon.StatusFail, err
	}
	if err := catalog.Close(); err != nil {
		return polochon.StatusFail, err
	}
	return polochon.StatusOK, nil
}
