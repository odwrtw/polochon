package yifysubtitles

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"path/filepath"

	"github.com/agnivade/levenshtein"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/yifysubs"
)

// Make sure that the module is a subtitler
var _ polochon.Subtitler = (*YifySubs)(nil)

func init() {
	polochon.RegisterModule(&YifySubs{})
}

// YifySubs holds the YifySubs module
type YifySubs struct {
	Client     Searcher
	log        *slog.Logger
	baseURL    string
	configured bool
}

// Module constants
const (
	moduleName = "yifysubs"
)

// Searcher is an interface to search subtitles
type Searcher interface {
	SearchByLang(imdbID, lang string) ([]*yifysubs.Subtitle, error)
}

// Errors
var (
	ErrInvalidSubtitleLang = errors.New("yifysub: invalid subtitle language")
	ErrMissingImdbID       = errors.New("yifysub: missing imdb id")
)

// Init implements the module interface
func (y *YifySubs) Init(p []byte, log *slog.Logger) error {
	if log == nil {
		log = slog.Default()
	}
	y.log = log.With("module", moduleName)
	if y.configured {
		return nil
	}

	c := yifysubs.NewDefault()
	y.Client = c
	y.baseURL = c.Endpoint
	y.configured = true
	return nil
}

// Name implements the Module interface
func (y *YifySubs) Name() string {
	return moduleName
}

// Status implements the Module interface
func (y *YifySubs) Status() (polochon.ModuleStatus, error) {
	_, err := y.Client.SearchByLang("tt0133093", "English")
	if err != nil {
		return polochon.StatusFail, err
	}

	return polochon.StatusOK, nil
}

func (y *YifySubs) getMovieSubtitle(m *polochon.Movie, lang polochon.Language) (*yifysubs.Subtitle, error) {
	if m.ImdbID == "" {
		return nil, ErrMissingImdbID
	}

	subLang, err := lang.Name()
	if err != nil {
		return nil, ErrInvalidSubtitleLang
	}

	// Get the subs for this movie
	subs, err := y.Client.SearchByLang(m.ImdbID, subLang)
	if err != nil {
		if err == yifysubs.ErrNoSubtitleFound {
			return nil, polochon.ErrNoSubtitleFound
		}

		return nil, err
	}

	var selected *yifysubs.Subtitle
	minScore := 1000

	release := filepath.Base(m.PathWithoutExt())
	for _, sub := range subs {
		for _, subRelease := range sub.Releases {
			dist := levenshtein.ComputeDistance(release, subRelease)
			if dist < minScore {
				selected = sub
				minScore = dist
			}
		}
	}

	if selected == nil {
		return nil, polochon.ErrNoSubtitleFound
	}

	return selected, nil
}

// GetMovieSubtitle will get a movie subtitle
func (y *YifySubs) GetMovieSubtitle(m *polochon.Movie, lang polochon.Language) (*polochon.Subtitle, error) {
	s, err := y.getMovieSubtitle(m, lang)
	if err != nil {
		return nil, err
	}

	data := &bytes.Buffer{}
	_, err = data.ReadFrom(s)
	if err != nil {
		return nil, err
	}

	sub := polochon.NewSubtitleFromVideo(m, lang)
	sub.Data = data.Bytes()
	return sub, nil
}

// ListSubtitles implements the Subtitler interface.
func (y *YifySubs) ListSubtitles(_ context.Context, i any, lang polochon.Language) ([]*polochon.SubtitleEntry, error) {
	m, ok := i.(*polochon.Movie)
	if !ok {
		return nil, polochon.ErrNotAvailable
	}

	if m.ImdbID == "" {
		return nil, ErrMissingImdbID
	}

	subLang, err := lang.Name()
	if err != nil {
		return nil, ErrInvalidSubtitleLang
	}

	subs, err := y.Client.SearchByLang(m.ImdbID, subLang)
	if err != nil {
		if err == yifysubs.ErrNoSubtitleFound {
			return nil, polochon.ErrNoSubtitleFound
		}
		return nil, err
	}

	var entries []*polochon.SubtitleEntry
	for _, sub := range subs {
		u, err := url.Parse(sub.URL)
		if err != nil {
			y.log.Warn("invalid subtitle URL", "url", sub.URL, "imdb_id", m.ImdbID)
			continue
		}
		for _, rel := range sub.Releases {
			entries = append(entries, &polochon.SubtitleEntry{
				Language:    lang,
				Description: rel,
				ID:          u.Path,
			})
		}
	}

	if len(entries) == 0 {
		return nil, polochon.ErrNoSubtitleFound
	}
	return entries, nil
}

// DownloadSubtitle implements the Subtitler interface.
func (y *YifySubs) DownloadSubtitle(_ context.Context, i any, entry *polochon.SubtitleEntry) (*polochon.Subtitle, error) {
	m, ok := i.(*polochon.Movie)
	if !ok {
		return nil, polochon.ErrNotAvailable
	}

	sub := &yifysubs.Subtitle{URL: y.baseURL + entry.ID}

	data := &bytes.Buffer{}
	if _, err := data.ReadFrom(sub); err != nil {
		return nil, err
	}

	s := polochon.NewSubtitleFromVideo(m, entry.Language)
	s.Data = data.Bytes()
	return s, nil
}

// GetSubtitle implements the Subtitler interface
func (y *YifySubs) GetSubtitle(_ context.Context, i any, lang polochon.Language) (*polochon.Subtitle, error) {
	switch v := i.(type) {
	case *polochon.Movie:
		return y.GetMovieSubtitle(v, lang)
	default:
		return nil, fmt.Errorf("yifysubs: can only search for movie subtitles")
	}
}
