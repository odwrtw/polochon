// Package x1337x implements the 1337x torrent source.
package x1337x

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/dustin/go-humanize"
	"github.com/goccy/go-yaml"
	"github.com/odwrtw/whatsthis"

	polochon "github.com/odwrtw/polochon/lib"
)

var (
	_ polochon.Torrenter = (*X1337)(nil)

	ErrMissingURLs              = errors.New("1337x: no endpoints configured")
	ErrInvalidEndpoint          = errors.New("1337x: invalid endpoint")
	ErrUnexpectedResponse       = errors.New("1337x: unexpected response")
	ErrNoWorkingEndpoint        = errors.New("1337x: no working endpoint")
	defaultTimeout              = 30 * time.Second
	maxResponseSize       int64 = 8 << 20
)

const moduleName = "1337x"

// Params configures the 1337x endpoints. URLs are tried in order; the last
// endpoint that returns a valid 1337x response is preferred on later calls.
// ReleaseGroups, when set, is an allow-list matched against the release group
// guessed from each result name.
type Params struct {
	URLs          []string `yaml:"urls"`
	Timeout       string   `yaml:"timeout"`
	ReleaseGroups []string `yaml:"release_groups"`
}

// X1337 is a source for movie and episode torrents.
type X1337 struct {
	Client  *http.Client
	Timeout time.Duration

	urls          []string
	releaseGroups map[string]struct{}
	preferred     int
	mu            sync.Mutex
	log           *slog.Logger
	configured    bool
}

func init() {
	polochon.RegisterModule(&X1337{})
}

// Init implements polochon.Module.
func (x *X1337) Init(p []byte, log *slog.Logger) error {
	if x.configured {
		return nil
	}
	if log == nil {
		log = slog.Default()
	}
	x.log = log.With("module", moduleName)

	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return err
	}
	return x.InitWithParams(params)
}

// InitWithParams configures the module.
func (x *X1337) InitWithParams(params *Params) error {
	if x.configured {
		return nil
	}
	if params == nil {
		return ErrMissingURLs
	}
	if x.log == nil {
		x.log = slog.Default().With("module", moduleName)
	}

	urls, err := normalizeURLs(params.URLs)
	if err != nil {
		return err
	}
	if len(urls) == 0 {
		return ErrMissingURLs
	}

	timeout := defaultTimeout
	if params.Timeout != "" {
		timeout, err = time.ParseDuration(params.Timeout)
		if err != nil {
			return err
		}
	}

	x.urls = urls
	x.releaseGroups = make(map[string]struct{}, len(params.ReleaseGroups))
	for _, group := range params.ReleaseGroups {
		if group = strings.TrimSpace(group); group != "" {
			x.releaseGroups[strings.ToLower(group)] = struct{}{}
		}
	}
	x.Timeout = timeout
	if x.Client == nil {
		x.Client = &http.Client{}
	}
	x.configured = true
	return nil
}

// Name implements polochon.Module.
func (x *X1337) Name() string { return moduleName }

// Status implements polochon.Module.
func (x *X1337) Status() (polochon.ModuleStatus, error) {
	torrents, err := x.SearchTorrents("black mirror")
	if err != nil || len(torrents) == 0 {
		return polochon.StatusFail, err
	}
	return polochon.StatusOK, nil
}

// GetTorrents implements polochon.Torrenter.
func (x *X1337) GetTorrents(ctx context.Context, value any) error {
	var (
		query  string
		accept func(whatsthis.Info) bool
		apply  func([]*polochon.Torrent)
	)

	switch v := value.(type) {
	case *polochon.Movie:
		query = v.Title
		accept = func(guess whatsthis.Info) bool {
			return guess.Type == whatsthis.Movie && strings.EqualFold(guess.Title, v.Title)
		}
		apply = func(torrents []*polochon.Torrent) {
			for _, torrent := range torrents {
				torrent.ImdbID = v.ImdbID
				torrent.Type = polochon.TypeMovie
			}
			v.Torrents = torrents
		}
	case *polochon.ShowEpisode:
		query = fmt.Sprintf("%s S%02dE%02d", v.ShowTitle, v.Season, v.Episode)
		accept = func(guess whatsthis.Info) bool {
			return guess.Type == whatsthis.Episode &&
				strings.EqualFold(guess.Title, v.ShowTitle) &&
				guess.Season == v.Season && guess.Episode == v.Episode
		}
		apply = func(torrents []*polochon.Torrent) {
			for _, torrent := range torrents {
				torrent.ImdbID = v.ShowImdbID
				torrent.Type = polochon.TypeEpisode
				torrent.Season = v.Season
				torrent.Episode = v.Episode
			}
			v.Torrents = torrents
		}
	default:
		return fmt.Errorf("1337x: invalid torrentable type %T", value)
	}

	results, err := x.search(ctx, query)
	if err != nil {
		return err
	}

	torrents := make([]*polochon.Torrent, 0, len(results))
	for _, result := range results {
		guess := whatsthis.Video(strings.ReplaceAll(result.Name, " ", "."))
		if !accept(guess) || !x.acceptsReleaseGroup(guess.ReleaseGroup) {
			continue
		}
		quality := qualityFromName(result.Name)
		if !quality.IsAllowed() {
			continue
		}
		torrents = append(torrents, result.polochonTorrent(quality))
	}
	torrents = polochon.FilterTorrents(torrents)
	if len(torrents) == 0 {
		return polochon.ErrTorrentNotFound
	}
	apply(torrents)
	return nil
}

// SearchTorrents implements polochon.Torrenter.
func (x *X1337) SearchTorrents(query string) ([]*polochon.Torrent, error) {
	results, err := x.search(context.Background(), query)
	if err != nil {
		return nil, err
	}

	torrents := make([]*polochon.Torrent, 0, len(results))
	for _, result := range results {
		torrents = append(torrents, result.polochonTorrent(qualityFromName(result.Name)))
	}
	return torrents, nil
}

type searchResult struct {
	Name     string
	Detail   string
	Seeders  int
	Leechers int
	User     string
	Size     int
	Magnet   string
}

func (r searchResult) polochonTorrent(quality polochon.Quality) *polochon.Torrent {
	return &polochon.Torrent{
		Quality: quality,
		Result: &polochon.TorrentResult{
			Name:       r.Name,
			URL:        r.Magnet,
			Seeders:    r.Seeders,
			Leechers:   r.Leechers,
			UploadUser: r.User,
			Size:       r.Size,
			Source:     moduleName,
		},
	}
}

// search tries the preferred endpoint first, then each configured fallback.
// A backend is considered healthy only after both its search page and at least
// one matching detail page have the expected 1337x structure.
func (x *X1337) search(ctx context.Context, query string) ([]searchResult, error) {
	if !x.configured || len(x.urls) == 0 {
		return nil, ErrMissingURLs
	}

	var errs []error
	for _, endpoint := range x.endpoints() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		requestCtx, cancel := context.WithTimeout(ctx, x.Timeout)
		results, err := x.searchEndpoint(requestCtx, endpoint, query)
		cancel()
		if err == nil {
			x.setPreferred(endpoint)
			return results, nil
		}
		x.log.Debug("1337x endpoint failed", "endpoint", endpoint, "error", err)
		errs = append(errs, fmt.Errorf("%s: %w", endpoint, err))
	}

	return nil, fmt.Errorf("%w: %w", ErrNoWorkingEndpoint, errors.Join(errs...))
}

func (x *X1337) searchEndpoint(ctx context.Context, endpoint, query string) ([]searchResult, error) {
	searchURL := endpoint + "/search/" + url.PathEscape(query) + "/1/"
	body, err := x.get(ctx, searchURL)
	if err != nil {
		return nil, err
	}

	results, err := parseSearchResults(body)
	if err != nil || len(results) == 0 {
		return results, err
	}

	magnets := 0
	for i := range results {
		detailURL, err := detailURL(endpoint, results[i].Detail)
		if err != nil {
			return nil, err
		}
		detail, err := x.get(ctx, detailURL)
		if err != nil {
			x.log.Debug("1337x detail request failed", "url", detailURL, "error", err)
			continue
		}
		results[i].Magnet = parseMagnet(detail)
		if results[i].Magnet != "" {
			magnets++
		}
	}
	if magnets == 0 {
		return nil, fmt.Errorf("%w: no magnet links", ErrUnexpectedResponse)
	}

	withMagnets := results[:0]
	for _, result := range results {
		if result.Magnet != "" {
			withMagnets = append(withMagnets, result)
		}
	}
	return withMagnets, nil
}

func (x *X1337) get(ctx context.Context, requestURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; polochon)")

	response, err := x.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected HTTP status %s", response.Status)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxResponseSize {
		return nil, fmt.Errorf("response exceeds %d bytes", maxResponseSize)
	}
	return body, nil
}

func parseSearchResults(body []byte) ([]searchResult, error) {
	document, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	if document.Find("table.table-list").Length() == 0 {
		return nil, ErrUnexpectedResponse
	}

	results := []searchResult{}
	document.Find("tr").Each(func(_ int, row *goquery.Selection) {
		var result searchResult
		row.Find("a").EachWithBreak(func(_ int, link *goquery.Selection) bool {
			href, ok := link.Attr("href")
			if !ok || !isTorrentPath(href) {
				return true
			}
			result.Name = strings.TrimSpace(link.Text())
			result.Detail = href
			return false
		})
		if result.Detail == "" {
			return
		}

		if strings.Contains(result.Name, "...") {
			result.Name = titleFromDetail(result.Detail, result.Name)
		}
		result.Seeders = intColumn(row, "coll-2")
		result.Leechers = intColumn(row, "coll-3")
		result.Size = sizeColumn(row)
		result.User = column(row, "coll-5")
		results = append(results, result)
	})
	return results, nil
}

func parseMagnet(body []byte) string {
	document, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return ""
	}
	magnet, _ := document.Find(`a[href^="magnet:"]`).First().Attr("href")
	return magnet
}

func column(row *goquery.Selection, classPrefix string) string {
	text := ""
	row.Find("td").EachWithBreak(func(_ int, cell *goquery.Selection) bool {
		class, _ := cell.Attr("class")
		if strings.HasPrefix(class, classPrefix) {
			text = strings.TrimSpace(cell.Text())
			return false
		}
		return true
	})
	return text
}

func intColumn(row *goquery.Selection, classPrefix string) int {
	value := strings.ReplaceAll(column(row, classPrefix), ",", "")
	result, _ := strconv.Atoi(value)
	return result
}

func sizeColumn(row *goquery.Selection) int {
	size, err := humanize.ParseBytes(column(row, "coll-4"))
	if err != nil || size > uint64(^uint(0)>>1) {
		return 0
	}
	return int(size)
}

func detailURL(endpoint, detail string) (string, error) {
	base, err := url.Parse(endpoint + "/")
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(detail)
	if err != nil {
		return "", err
	}
	resolved := base.ResolveReference(ref)
	if resolved.Scheme != base.Scheme || resolved.Host != base.Host || !strings.HasPrefix(resolved.Path, "/torrent/") {
		return "", fmt.Errorf("%w: invalid detail URL", ErrUnexpectedResponse)
	}
	return resolved.String(), nil
}

func isTorrentPath(href string) bool {
	parsed, err := url.Parse(href)
	return err == nil && strings.HasPrefix(parsed.Path, "/torrent/")
}

func titleFromDetail(detail, fallback string) string {
	parsed, err := url.Parse(detail)
	if err != nil {
		return fallback
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 3 {
		return fallback
	}
	title, err := url.PathUnescape(parts[2])
	if err != nil {
		return fallback
	}
	return strings.ReplaceAll(title, "-", " ")
}

func (x *X1337) acceptsReleaseGroup(group string) bool {
	if len(x.releaseGroups) == 0 {
		return true
	}
	_, ok := x.releaseGroups[strings.ToLower(strings.TrimSpace(group))]
	return ok
}

func qualityFromName(name string) polochon.Quality {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "2160p"), strings.Contains(lower, "4k"):
		return polochon.Quality2160p
	case strings.Contains(lower, "1080p"):
		return polochon.Quality1080p
	case strings.Contains(lower, "720p"):
		return polochon.Quality720p
	case strings.Contains(lower, "480p"):
		return polochon.Quality480p
	case strings.Contains(lower, "3d"):
		return polochon.Quality3D
	default:
		return polochon.Quality("")
	}
}

func normalizeURLs(urls []string) ([]string, error) {
	result := make([]string, 0, len(urls))
	seen := make(map[string]struct{}, len(urls))
	for _, raw := range urls {
		parsed, err := url.ParseRequestURI(raw)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
			parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
			return nil, fmt.Errorf("%w: %q", ErrInvalidEndpoint, raw)
		}
		normalized := strings.TrimRight(parsed.String(), "/")
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result, nil
}

func (x *X1337) endpoints() []string {
	x.mu.Lock()
	defer x.mu.Unlock()

	endpoints := make([]string, 0, len(x.urls))
	for offset := range x.urls {
		endpoints = append(endpoints, x.urls[(x.preferred+offset)%len(x.urls)])
	}
	return endpoints
}

func (x *X1337) setPreferred(endpoint string) {
	x.mu.Lock()
	defer x.mu.Unlock()
	for i, candidate := range x.urls {
		if candidate == endpoint {
			x.preferred = i
			return
		}
	}
}
