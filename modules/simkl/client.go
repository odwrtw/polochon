package simkl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	_ "time/tzdata"

	"golang.org/x/sync/singleflight"
)

const maxBody = 16 << 20

// APIError exposes machine-readable provider failures without upstream bodies or secrets.
type APIError struct {
	Status  int
	Code    string
	RetryAt time.Time
}

func (e *APIError) Error() string {
	return fmt.Sprintf("simkl: upstream HTTP %d (%s)", e.Status, e.Code)
}
func (e *APIError) Is(target error) bool {
	switch target {
	case ErrQuota:
		return e.Status == 429 || e.Code == "user_limit_exceeded" || e.Code == "app_limit_exceeded" || e.Code == "RATE_LIMIT"
	case ErrScope:
		return e.Code == "insufficient_scope" || e.Code == "invalid_scope"
	case ErrPlan:
		return e.Code == "pro_required" || e.Code == "vip_required"
	case ErrCredentials:
		return e.Code == "invalid_grant" || e.Code == "invalid_client" || e.Code == "expired_token"
	case ErrConnectionRequired:
		return e.Code == "user_token_required"
	case ErrNotFound:
		return e.Status == 404
	}
	return false
}

type cacheEntry struct {
	body                []byte
	etag, modified      string
	expires, staleUntil time.Time
}

type client struct {
	params            Params
	http              *http.Client
	api, cdn, version string
	now               func() time.Time
	sleep             func(context.Context, time.Duration) error
	pace              time.Duration
	gate, authGate    chan struct{}
	next, postNext    time.Time // protected by gate
	mu                sync.Mutex
	cache             map[string]cacheEntry
	cacheBytes        int
	mappings          map[string]map[string]struct{}
	flights           singleflight.Group
	access            string // protected by authGate
	expires           time.Time
	authErr           error
	authRetry         time.Time
	quotaUntil        time.Time // protected by mu
}

func newClient(p Params) *client {
	return &client{params: p, http: newHTTPClient(), api: "https://api.simkl.com", cdn: "https://data.simkl.in", version: appVersion(), now: time.Now, sleep: waitContext, pace: 150 * time.Millisecond,
		gate: make(chan struct{}, 1), authGate: make(chan struct{}, 1), cache: make(map[string]cacheEntry), mappings: make(map[string]map[string]struct{})}
}

func waitContext(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func acquire(ctx context.Context, gate chan struct{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case gate <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *client) endpoint(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", ErrUnsupported
	}
	q := u.Query()
	q.Set("app-name", "polochon")
	q.Set("app-version", c.version)
	if c.params.ClientID != "" {
		q.Set("client_id", c.params.ClientID)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// raw serializes HTTP exchanges, but never holds the gate across token refresh.
func (c *client) raw(ctx context.Context, method, target, token string, form url.Values, headers http.Header) ([]byte, http.Header, int, error) {
	if err := acquire(ctx, c.gate); err != nil {
		return nil, nil, 0, err
	}
	defer func() { <-c.gate }()
	if token != "" {
		c.mu.Lock()
		quota := c.quotaUntil
		c.mu.Unlock()
		if c.now().Before(quota) {
			return nil, nil, 429, &APIError{Status: 429, Code: "user_limit_exceeded", RetryAt: quota}
		}
	}
	when := c.next
	if method == http.MethodPost && c.postNext.After(when) {
		when = c.postNext
	}
	if err := c.sleep(ctx, when.Sub(c.now())); err != nil {
		return nil, nil, 0, err
	}
	target, err := c.endpoint(target)
	if err != nil {
		return nil, nil, 0, err
	}
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, nil, 0, ErrUnsupported
	}
	req.Header.Set("User-Agent", "polochon/"+c.version+" (https://github.com/odwrtw/polochon)")
	req.Header.Set("Accept", "application/json")
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for k, values := range headers {
		req.Header[k] = append([]string(nil), values...)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	c.next = c.now().Add(c.pace)
	if method == http.MethodPost {
		c.postNext = c.now().Add(time.Second)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, nil, 0, ctx.Err()
		}
		// net/url.Error includes the request URL; never return it with credentials.
		return nil, nil, 0, errors.New("simkl: upstream transport failure")
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil {
		return nil, nil, resp.StatusCode, errors.New("simkl: incomplete response body")
	}
	if len(data) > maxBody {
		return nil, nil, resp.StatusCode, errors.New("simkl: response exceeds size limit")
	}
	return data, resp.Header, resp.StatusCode, nil
}

func safeCode(code string) string {
	switch code {
	case "rate_limit", "user_limit_exceeded", "app_limit_exceeded", "RATE_LIMIT", "client_id_failed", "user_token_required", "user_token_failed", "invalid_grant", "invalid_client", "expired_token", "insufficient_scope", "invalid_scope", "pro_required", "vip_required", "max_items", "authorization_pending", "slow_down", "access_denied", "invalid_request", "server_error":
		return code
	}
	return "upstream_error"
}

func responseError(data []byte, h http.Header, status int, now time.Time) error {
	var envelope struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(data, &envelope)
	if status >= 200 && status < 300 && envelope.Error == "" {
		return nil
	}
	e := &APIError{Status: status, Code: safeCode(envelope.Error)}
	if n, err := strconv.Atoi(h.Get("Retry-After")); err == nil && n > 0 {
		e.RetryAt = now.Add(time.Duration(n) * time.Second)
	} else if t, err := http.ParseTime(h.Get("Retry-After")); err == nil {
		e.RetryAt = t
	}
	if e.Code == "user_limit_exceeded" || e.Code == "app_limit_exceeded" {
		reset := quotaReset(now)
		if reset.After(e.RetryAt) {
			e.RetryAt = reset
		}
	}
	// Burst-limit Retry-After can misleadingly contain the daily reset.
	if e.Code == "rate_limit" {
		e.RetryAt = time.Time{}
	}
	return e
}

func quotaReset(now time.Time) time.Time {
	loc, _ := time.LoadLocation("America/New_York")
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, loc)
}

func (c *client) get(ctx context.Context, target string, auth bool, ttl time.Duration, stale bool, out any) error {
	key := target
	if auth {
		key = "private:" + key
	}
	c.mu.Lock()
	entry := c.cache[key]
	c.mu.Unlock()
	if entry.body != nil && c.now().Before(entry.expires) {
		return json.Unmarshal(entry.body, out)
	}
	result := c.flights.DoChan(key, func() (any, error) { return c.fetch(ctx, target, key, auth, ttl, stale) })
	select {
	case <-ctx.Done():
		return ctx.Err()
	case r := <-result:
		if r.Err != nil {
			return r.Err
		}
		return json.Unmarshal(r.Val.([]byte), out)
	}
}

func (c *client) fetch(ctx context.Context, target, key string, auth bool, ttl time.Duration, stale bool) ([]byte, error) {
	c.mu.Lock()
	entry := c.cache[key]
	quota := c.quotaUntil
	c.mu.Unlock()
	if entry.body != nil && c.now().Before(entry.expires) {
		return entry.body, nil
	}
	if auth && c.now().Before(quota) {
		return nil, &APIError{Status: 429, Code: "user_limit_exceeded", RetryAt: quota}
	}
	headers := make(http.Header)
	if entry.etag != "" {
		headers.Set("If-None-Match", entry.etag)
	}
	if entry.modified != "" {
		headers.Set("If-Modified-Since", entry.modified)
	}
	var last error
	refreshed := false
	for attempt := 0; attempt < 3; attempt++ {
		token := ""
		var err error
		if auth {
			token, err = c.accessToken(ctx, "")
			if err != nil {
				return nil, err
			}
		}
		data, h, status, err := c.raw(ctx, http.MethodGet, target, token, nil, headers)
		if err == nil && status == http.StatusNotModified && entry.body != nil {
			data = entry.body
			if h.Get("ETag") == "" {
				h.Set("ETag", entry.etag)
			}
			if h.Get("Last-Modified") == "" {
				h.Set("Last-Modified", entry.modified)
			}
		} else if err == nil {
			err = responseError(data, h, status, c.now())
		}
		var apiErr *APIError
		if auth && errors.As(err, &apiErr) && status == 401 && (apiErr.Code == "expired_token" || apiErr.Code == "user_token_failed" || apiErr.Code == "upstream_error") && !refreshed {
			if _, err = c.accessToken(ctx, token); err != nil {
				return nil, err
			}
			refreshed = true
			attempt--
			continue
		}
		if err == nil {
			if !json.Valid(data) {
				return nil, errors.New("simkl: malformed JSON response")
			}
			if auth && h.Get("X-RateLimit-Remaining") == "0" {
				c.mu.Lock()
				c.quotaUntil = quotaReset(c.now())
				c.mu.Unlock()
			}
			c.store(key, data, h, ttl, stale)
			return data, nil
		}
		last = err
		if errors.As(err, &apiErr) {
			if auth && (apiErr.Code == "user_limit_exceeded" || apiErr.Code == "app_limit_exceeded") {
				c.mu.Lock()
				c.quotaUntil = apiErr.RetryAt
				c.mu.Unlock()
				return nil, err
			}
			if status != 429 && status < 500 {
				return nil, err
			}
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if attempt < 2 {
			delay := time.Duration(1<<attempt)*time.Second + time.Duration(rand.IntN(250))*time.Millisecond
			if apiErr != nil && apiErr.RetryAt.After(c.now().Add(delay)) {
				delay = apiErr.RetryAt.Sub(c.now())
			}
			if delay > time.Minute {
				break
			}
			if err = c.sleep(ctx, delay); err != nil {
				return nil, err
			}
		}
	}
	if !auth && stale && entry.body != nil && c.now().Before(entry.staleUntil) {
		return entry.body, nil
	}
	return nil, last
}

func (c *client) store(key string, data []byte, h http.Header, ttl time.Duration, stale bool) {
	control := h.Get("Cache-Control")
	if strings.Contains(control, "no-store") {
		c.mu.Lock()
		if old, ok := c.cache[key]; ok {
			delete(c.cache, key)
			c.cacheBytes -= len(old.body)
		}
		c.mu.Unlock()
		return
	}
	for _, part := range strings.Split(control, ",") {
		part = strings.TrimSpace(part)
		if value, ok := strings.CutPrefix(part, "max-age="); ok {
			if seconds, err := strconv.Atoi(strings.Trim(value, "\"")); err == nil && seconds >= 0 {
				ttl = time.Duration(seconds) * time.Second
			}
		}
	}
	if strings.Contains(control, "no-cache") {
		ttl = 0
		stale = false
	}
	if age, err := strconv.Atoi(h.Get("Age")); err == nil && age > 0 {
		ttl -= time.Duration(age) * time.Second
		if ttl < 0 {
			ttl = 0
		}
	}
	if ttl > 24*time.Hour {
		ttl = 24 * time.Hour
	}
	entry := cacheEntry{body: bytes.Clone(data), etag: h.Get("ETag"), modified: h.Get("Last-Modified"), expires: c.now().Add(ttl)}
	entry.staleUntil = entry.expires
	if stale && !strings.Contains(control, "must-revalidate") {
		entry.staleUntil = entry.expires.Add(24 * time.Hour)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if old, ok := c.cache[key]; ok {
		c.cacheBytes -= len(old.body)
		delete(c.cache, key)
	}
	for len(c.cache) >= 128 || c.cacheBytes+len(data) > 32<<20 {
		for k, v := range c.cache {
			delete(c.cache, k)
			c.cacheBytes -= len(v.body)
			break
		}
	}
	c.cache[key] = entry
	c.cacheBytes += len(data)
}
