package tvdb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultAPIURL = "https://api4.thetvdb.com/v4"
	tokenLifetime = 28 * 24 * time.Hour
)

type apiClient struct {
	baseURL   string
	http      *http.Client
	apiKey    string
	pin       string
	token     string
	tokenTime time.Time
	mu        sync.Mutex
}

type apiError struct {
	statusCode int
	body       string
}

func (e *apiError) Error() string {
	if e.body == "" {
		return fmt.Sprintf("tvdb: API returned HTTP %d", e.statusCode)
	}
	return fmt.Sprintf("tvdb: API returned HTTP %d: %s", e.statusCode, e.body)
}

type alias struct {
	Name string `json:"name"`
}

type remoteID struct {
	ID         string `json:"id"`
	SourceName string `json:"sourceName"`
}

type artwork struct {
	Image string  `json:"image"`
	Score float64 `json:"score"`
	Type  int     `json:"type"`
}

type series struct {
	Aliases           []alias    `json:"aliases"`
	Artworks          []artwork  `json:"artworks"`
	AverageRuntime    *int       `json:"averageRuntime"`
	FirstAired        string     `json:"firstAired"`
	ID                int        `json:"id"`
	Image             string     `json:"image"`
	Name              string     `json:"name"`
	Overview          string     `json:"overview"`
	RemoteIDs         []remoteID `json:"remoteIds"`
	Score             float64    `json:"score"`
	Year              string     `json:"year"`
	DefaultSeasonType int        `json:"defaultSeasonType"`
}

type episode struct {
	Aired        string     `json:"aired"`
	ID           int        `json:"id"`
	Image        string     `json:"image"`
	Name         string     `json:"name"`
	Number       int        `json:"number"`
	Overview     string     `json:"overview"`
	RemoteIDs    []remoteID `json:"remoteIds"`
	Runtime      *int       `json:"runtime"`
	SeasonNumber int        `json:"seasonNumber"`
}

type searchResult struct {
	Aliases []string `json:"aliases"`
	Name    string   `json:"name"`
	TVDBID  string   `json:"tvdb_id"`
	Year    string   `json:"year"`
}

type response[T any] struct {
	Data   T      `json:"data"`
	Status string `json:"status"`
	Links  links  `json:"links"`
}

type links struct {
	Next string `json:"next"`
}

func newAPIClient(apiKey, pin string) *apiClient {
	return &apiClient{
		baseURL: defaultAPIURL,
		http:    &http.Client{Timeout: 30 * time.Second},
		apiKey:  apiKey,
		pin:     pin,
	}
}

func (c *apiClient) login(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Since(c.tokenTime) < tokenLifetime {
		return nil
	}

	payload := struct {
		APIKey string `json:"apikey"`
		Pin    string `json:"pin,omitempty"`
	}{
		APIKey: c.apiKey,
		Pin:    c.pin,
	}
	var result response[struct {
		Token string `json:"token"`
	}]
	if err := c.request(ctx, http.MethodPost, "/login", nil, payload, &result, false); err != nil {
		return err
	}
	if result.Data.Token == "" {
		return errors.New("tvdb: login response did not contain a token")
	}

	c.token = result.Data.Token
	c.tokenTime = time.Now()
	return nil
}

func (c *apiClient) searchSeries(ctx context.Context, query string, year int) ([]series, error) {
	params := url.Values{
		"query": {query},
		"type":  {"series"},
	}
	if year != 0 {
		params.Set("year", strconv.Itoa(year))
	}

	var result response[[]searchResult]
	if err := c.get(ctx, "/search", params, &result); err != nil {
		return nil, err
	}

	shows := make([]series, 0, len(result.Data))
	for _, item := range result.Data {
		id, err := strconv.Atoi(item.TVDBID)
		if err != nil || id == 0 {
			continue
		}
		show := series{ID: id, Name: item.Name, Year: item.Year}
		for _, name := range item.Aliases {
			show.Aliases = append(show.Aliases, alias{Name: name})
		}
		shows = append(shows, show)
	}
	return shows, nil
}

func (c *apiClient) searchSeriesByRemoteID(ctx context.Context, id string) ([]series, error) {
	var result response[[]struct {
		Series *series `json:"series"`
	}]
	path := "/search/remoteid/" + url.PathEscape(id)
	if err := c.get(ctx, path, nil, &result); err != nil {
		return nil, err
	}

	shows := make([]series, 0, len(result.Data))
	for _, item := range result.Data {
		if item.Series != nil {
			shows = append(shows, *item.Series)
		}
	}
	return shows, nil
}

func (c *apiClient) getSeries(ctx context.Context, id int) (*series, error) {
	var result response[series]
	path := fmt.Sprintf("/series/%d/extended", id)
	if err := c.get(ctx, path, nil, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

func (c *apiClient) getEpisodes(ctx context.Context, id, season, number int) ([]episode, error) {
	var episodes []episode
	for page := 0; ; page++ {
		params := url.Values{"page": {strconv.Itoa(page)}}
		if season != 0 {
			params.Set("season", strconv.Itoa(season))
		}
		if number != 0 {
			params.Set("episodeNumber", strconv.Itoa(number))
		}

		var result response[struct {
			Episodes []episode `json:"episodes"`
		}]
		path := fmt.Sprintf("/series/%d/episodes/default", id)
		if err := c.get(ctx, path, params, &result); err != nil {
			return nil, err
		}
		episodes = append(episodes, result.Data.Episodes...)
		if result.Links.Next == "" {
			return episodes, nil
		}
	}
}

func (c *apiClient) get(ctx context.Context, path string, params url.Values, dst any) error {
	if err := c.login(ctx); err != nil {
		return err
	}
	return c.request(ctx, http.MethodGet, path, params, nil, dst, true)
}

func (c *apiClient) request(
	ctx context.Context,
	method, path string,
	params url.Values,
	body, dst any,
	authenticated bool,
) error {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return err
	}
	u.RawQuery = params.Encode()

	var requestBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		requestBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), requestBody)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if authenticated {
		c.mu.Lock()
		token := c.token
		c.mu.Unlock()
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &apiError{statusCode: resp.StatusCode, body: strings.TrimSpace(string(data))}
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return err
	}
	return nil
}
