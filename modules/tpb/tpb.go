package tpb

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"

	yaml "github.com/goccy/go-yaml"
	"github.com/odwrtw/tpb"
	"github.com/odwrtw/whatsthis"

	polochon "github.com/odwrtw/polochon/lib"
)

// Make sure that the module is a torrenter and an explorer
var (
	_ polochon.Torrenter = (*TPB)(nil)
	_ polochon.Explorer  = (*TPB)(nil)
)

var (
	defaultTimeout = 30 * time.Second
)

func init() {
	polochon.RegisterModule(&TPB{})
}

// Module constants
const (
	moduleName = "thepiratebay"
)

// Params represents the module params
type Params struct {
	URLs       []string `yaml:"urls"`
	Timeout    string   `yaml:"timeout"`
	ShowUsers  []string `yaml:"show_users"`
	MovieUsers []string `yaml:"movie_users"`
}

// TPB is a source for torrents
type TPB struct {
	Client     *tpb.Client
	Timeout    time.Duration
	MovieUsers []string
	ShowUsers  []string
	log        *slog.Logger
	configured bool
}

// Init implements the module interface
func (t *TPB) Init(p []byte, log *slog.Logger) error {
	if log == nil {
		log = slog.Default()
	}
	t.log = log.With("module", moduleName)

	if t.configured {
		return nil
	}

	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return err
	}

	return t.InitWithParams(params)
}

// InitWithParams configures the module
func (t *TPB) InitWithParams(params *Params) error {
	if t.configured {
		return nil
	}

	if t.log == nil {
		t.log = slog.Default().With("module", moduleName)
	}

	t.Client = tpb.New(params.URLs...)
	t.MovieUsers = params.MovieUsers
	t.ShowUsers = params.ShowUsers
	t.configured = true

	var err error
	if params.Timeout == "" {
		t.Timeout = defaultTimeout
	} else {
		t.Timeout, err = time.ParseDuration(params.Timeout)
	}

	return err
}

// Name implements the Module interface
func (t *TPB) Name() string {
	return moduleName
}

// Status implements the Module interface
func (t *TPB) Status() (polochon.ModuleStatus, error) {
	torrents, err := t.SearchTorrents("black-mirror")
	if err != nil {
		return polochon.StatusFail, err
	}

	if len(torrents) == 0 {
		return polochon.StatusFail, nil
	}

	return polochon.StatusOK, nil
}

// searcher represents an interface to search torrent
type searcher interface {
	key() string
	users() []string
	defaultQuality() string
	setTorrents([]*polochon.Torrent)
	isValidGuess(guess whatsthis.Info) bool
	imdbID() string
}

// newSearcher will return a new Searcher
func (t *TPB) newSearcher(i any) (searcher, error) {
	switch v := i.(type) {
	case *polochon.ShowEpisode:
		return &showSearcher{
			Episode: v,
			Users:   t.ShowUsers,
		}, nil
	case *polochon.Movie:
		return &movieSearcher{
			Movie: v,
			Users: t.MovieUsers,
		}, nil
	default:
		return nil, fmt.Errorf("unknown type")
	}
}

// GetTorrents implements the Torrenter interface
func (t *TPB) search(s string) ([]*tpb.Torrent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), t.Timeout)
	defer cancel()

	return t.Client.Search(ctx, s, &tpb.SearchOptions{
		Category: tpb.Video,
	})
}

// GetTorrents implements the Torrenter interface
func (t *TPB) GetTorrents(_ context.Context, i any) error {
	// Create a new Searcher
	searcher, err := t.newSearcher(i)
	if err != nil {
		return err
	}

	torrents, err := t.search(searcher.key())
	if err != nil {
		return err
	}

	// Transform and filter the torrents we found
	pTorrents := t.transformTorrents(searcher, torrents)

	// Set the torrents into the video object
	searcher.setTorrents(pTorrents)
	return nil
}

// SearchTorrents implements the Torrenter interface
func (t *TPB) SearchTorrents(s string) ([]*polochon.Torrent, error) {
	results, err := t.search(s)
	if err != nil {
		return nil, err
	}

	torrents := make([]*polochon.Torrent, len(results))
	for i, t := range results {
		torrents[i] = &polochon.Torrent{
			Quality: getQuality(t.Name),
			Result: &polochon.TorrentResult{
				Name:       t.Name,
				URL:        t.Magnet(),
				Seeders:    t.Seeders,
				Leechers:   t.Leechers,
				Source:     moduleName,
				UploadUser: t.User,
				Size:       int(t.Size),
			},
		}
	}

	return torrents, nil
}

func filterTorrents(torrents []*tpb.Torrent, allowedUsers []string) []*tpb.Torrent {
	filteredList := []*tpb.Torrent{}
	if len(torrents) == 0 || len(allowedUsers) == 0 {
		return filteredList
	}

	// Create a set of users
	userMap := map[string]struct{}{}
	for _, u := range allowedUsers {
		userMap[u] = struct{}{}
	}

	for _, t := range torrents {
		if _, ok := userMap[t.User]; ok {
			filteredList = append(filteredList, t)
		}
	}

	return filteredList
}

// transformTorrents will filter and transform tpb.Torrent in polochon.Torrent
func (t *TPB) transformTorrents(s searcher, list []*tpb.Torrent) []*polochon.Torrent {
	torrents := []*polochon.Torrent{}
	for _, torrent := range filterTorrents(list, s.users()) {
		guess := whatsthis.File(torrent.Name)

		// Check the guess validity
		if !s.isValidGuess(guess) {
			continue
		}

		// If the torrent has an ImdbID, check it
		if torrent.ImdbID != "" && torrent.ImdbID != s.imdbID() {
			t.log.Debug("tpb: imdbIDs doesn't match", "torrent_imdb", torrent.ImdbID, "expected_imdb", s.imdbID())
			continue
		}

		// Set the default quality if none is defined
		torrentQuality := qualityFromGuess(torrent.Name, guess)
		if torrentQuality == "" {
			t.log.Debug("tpb: default quality", "torrent", torrent.Name)
			torrentQuality = polochon.Quality(s.defaultQuality())
		}

		// Check that the Quality is valid
		if !torrentQuality.IsAllowed() {
			t.log.Debug("tpb: unhandled quality", "quality", torrentQuality)
			continue
		}

		t.log.Debug("Adding torrent to the list", "torrent_quality", torrentQuality, "torrent_name", torrent.Name)

		torrents = append(torrents, &polochon.Torrent{
			Quality: torrentQuality,
			Result: &polochon.TorrentResult{
				Name:       torrent.Name,
				URL:        torrent.Magnet(),
				Seeders:    torrent.Seeders,
				Leechers:   torrent.Leechers,
				Source:     moduleName,
				UploadUser: torrent.User,
				Size:       int(torrent.Size),
			},
		})
	}
	// Filter the torrents to keep only the best ones
	return polochon.FilterTorrents(torrents)
}

func getQuality(name string) polochon.Quality {
	quality := qualityFromGuess(name, whatsthis.File(name))
	if !quality.IsAllowed() {
		return polochon.Quality("unknown")
	}
	return quality
}

func qualityFromGuess(name string, guess whatsthis.Info) polochon.Quality {
	if guess.ScreenSize != "" {
		// Preserve unsupported resolutions so callers cannot mistake them for
		// missing metadata and apply a lower-resolution default.
		return polochon.Quality(guess.ScreenSize)
	}
	// whatsthis has no presentation field. Retain the legacy 3D quality only
	// for a standalone marker that is neither title text nor a release credit.
	if has3DToken(name) && !has3DToken(guess.Title) && !has3DToken(guess.ReleaseGroup) {
		return polochon.Quality3D
	}
	return ""
}

func has3DToken(s string) bool {
	for _, token := range strings.FieldsFunc(s, func(r rune) bool {
		// Keep compact hyphenated credits (x264-3D) out of the fallback.
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-'
	}) {
		if strings.EqualFold(token, "3D") {
			return true
		}
	}
	return false
}
