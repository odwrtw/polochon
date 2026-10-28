package simkl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAPIURL  = "https://api.simkl.com"
	defaultDataURL = "https://data.simkl.in"
	maxBodySize    = 16 << 20
)

type apiClient struct {
	baseURL    string
	dataURL    string
	http       *http.Client
	clientID   string
	appVersion string
	gate       chan struct{}
	interval   time.Duration
	next       time.Time // guarded by gate; uncached calls stay sequential
}

type apiError struct {
	statusCode int
	body       string
	retryAfter string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("simkl: HTTP %d: %s (Retry-After: %s)", e.statusCode, e.body, e.retryAfter)
}

func newAPIClient(clientID, version string) *apiClient {
	return &apiClient{
		baseURL: defaultAPIURL, dataURL: defaultDataURL,
		http:     &http.Client{Timeout: 30 * time.Second},
		clientID: clientID, appVersion: version,
		gate: make(chan struct{}, 1), interval: 100 * time.Millisecond,
	}
}

// fetch never follows redirects. In particular, /redirect returns an ID in its
// Location header, not JSON; following it would fetch a website instead of API data.
func (c *apiClient) fetch(ctx context.Context, base, path string, params url.Values, dynamic bool) ([]byte, http.Header, int, error) {
	if dynamic {
		select {
		case c.gate <- struct{}{}:
		case <-ctx.Done():
			return nil, nil, 0, ctx.Err()
		}
		defer func() {
			c.next = time.Now().Add(c.interval)
			<-c.gate
		}()
		timer := time.NewTimer(time.Until(c.next))
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return nil, nil, 0, ctx.Err()
		}
	}

	u, err := url.Parse(base + path)
	if err != nil {
		return nil, nil, 0, err
	}
	query := maps.Clone(params)
	if query == nil {
		query = url.Values{}
	}
	query.Set("client_id", c.clientID)
	query.Set("app-name", "polochon")
	query.Set("app-version", c.appVersion)
	u.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "polochon/"+c.appVersion)
	client := *c.http
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusMovedPermanently {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, nil, 0, &apiError{
			statusCode: resp.StatusCode, body: strings.TrimSpace(string(body)),
			retryAfter: resp.Header.Get("Retry-After"),
		}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize+1))
	if err != nil {
		return nil, nil, 0, err
	}
	if len(data) > maxBodySize {
		return nil, nil, 0, ErrInvalidResponse
	}
	return data, resp.Header, resp.StatusCode, nil
}

func (c *apiClient) get(ctx context.Context, base, path string, params url.Values, dynamic bool, dst any) error {
	data, _, status, err := c.fetch(ctx, base, path, params, dynamic)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return ErrInvalidResponse
	}
	return json.Unmarshal(data, dst)
}

func (c *apiClient) search(ctx context.Context, kind, query string) ([]listing, error) {
	var items []listing
	params := url.Values{"q": {query}, "page": {"1"}, "limit": {"20"}}
	err := c.get(ctx, c.baseURL, "/search/"+kind, params, true, &items)
	return items, err
}

func (c *apiClient) detail(ctx context.Context, kind string, id int) (*mediaDetail, error) {
	if id <= 0 {
		return nil, ErrInvalidArgument
	}
	var raw json.RawMessage
	if err := c.get(ctx, c.baseURL, "/"+kind+"/"+strconv.Itoa(id), nil, false, &raw); err != nil {
		return nil, err
	}
	// Unknown numeric IDs return 200 [] rather than 404 (Simkl null-values docs).
	raw = bytes.TrimSpace(raw)
	if bytes.Equal(raw, []byte("null")) {
		return nil, ErrNotFound
	}
	if len(raw) > 0 && raw[0] == '[' {
		var items []json.RawMessage
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return nil, ErrNotFound
		}
		return nil, ErrInvalidResponse
	}
	var item mediaDetail
	if err := json.Unmarshal(raw, &item); err != nil {
		return nil, err
	}
	wantType := "show"
	if kind == "movies" {
		wantType = "movie"
	}
	if item.IDs.id() != id || item.Title == "" || item.Type != wantType {
		return nil, ErrInvalidResponse
	}
	return &item, nil
}

func (c *apiClient) resolve(ctx context.Context, kind string, params url.Values) (int, error) {
	params.Set("to", "simkl")
	mediaType := "tv"
	if kind == "movies" {
		mediaType = "movie"
	}
	params.Set("type", mediaType)
	_, headers, status, err := c.fetch(ctx, c.baseURL, "/redirect", params, true)
	if err != nil {
		return 0, err
	}
	if status != http.StatusMovedPermanently {
		return 0, ErrInvalidResponse
	}
	location, err := url.Parse(headers.Get("Location"))
	if err != nil || location.Host != "simkl.com" || location.User != nil ||
		(location.Scheme != "" && location.Scheme != "https") {
		return 0, ErrNotFound
	}
	parts := strings.Split(strings.Trim(location.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != kind {
		return 0, ErrNotFound
	}
	id, err := strconv.Atoi(parts[1])
	if err != nil || id <= 0 {
		return 0, ErrNotFound
	}
	return id, nil
}
