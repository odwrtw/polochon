package simkl

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
}

func (c *client) form() url.Values {
	f := url.Values{"client_id": {c.params.ClientID}}
	if c.params.ClientSecret != "" {
		f.Set("client_secret", c.params.ClientSecret)
	}
	return f
}

func (c *client) post(ctx context.Context, path string, form url.Values, out any) error {
	data, h, status, err := c.raw(ctx, http.MethodPost, c.api+path, "", form, nil)
	if err != nil {
		return err
	}
	if err = responseError(data, h, status, c.now()); err != nil {
		return err
	}
	if err = json.Unmarshal(data, out); err != nil {
		return errors.New("simkl: malformed authorization response")
	}
	return nil
}

func validToken(t tokenResponse) error {
	if !hasScope(t.Scope) {
		return ErrScope
	}
	if t.AccessToken == "" || t.RefreshToken == "" || t.ExpiresIn <= 0 || t.ExpiresIn > 366*86400 || (t.TokenType != "Bearer" && t.TokenType != "bearer") {
		return errors.New("simkl: invalid token response")
	}
	return nil
}

func (c *client) accessToken(ctx context.Context, rejected string) (string, error) {
	if c.params.ClientID == "" || c.params.RefreshToken == "" {
		return "", ErrConnectionRequired
	}
	if err := acquire(ctx, c.authGate); err != nil {
		return "", err
	}
	defer func() { <-c.authGate }()
	// A concurrent caller may already have replaced the rejected token.
	if c.access != "" && c.access != rejected && c.now().Before(c.expires) {
		return c.access, nil
	}
	if c.authErr != nil && c.now().Before(c.authRetry) {
		return "", c.authErr
	}
	f := c.form()
	f.Set("grant_type", "refresh_token")
	f.Set("refresh_token", c.params.RefreshToken)
	var t tokenResponse
	err := c.post(ctx, "/oauth2/token", f, &t)
	if err == nil {
		err = validToken(t)
	}
	if err == nil && t.RefreshToken != c.params.RefreshToken {
		err = errors.New("simkl: refresh token rotated; fixed-configuration authentication is unsupported")
	}
	if err != nil {
		if ctx.Err() == nil {
			c.authErr = err
			c.authRetry = c.now().Add(time.Minute)
		}
		return "", err
	}
	margin := time.Duration(t.ExpiresIn) * time.Second / 10
	if margin > 24*time.Hour {
		margin = 24 * time.Hour
	}
	c.access = t.AccessToken
	c.expires = c.now().Add(time.Duration(t.ExpiresIn)*time.Second - margin)
	c.authErr = nil
	return c.access, nil
}

// DeviceAuthorization contains the public instructions and the private polling credential.
// DeviceCode must never be displayed or logged.
type DeviceAuthorization struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int64  `json:"expires_in"`
	Interval                int64  `json:"interval"`
}

// AuthorizeDevice performs the read-only device flow and verifies fixed-token refresh
// with two fresh clients, simulating independent cold starts. It never writes a file.
func AuthorizeDevice(ctx context.Context, p Params, show func(DeviceAuthorization) error) (string, error) {
	return authorizeDevice(ctx, newClient(p), show)
}

func authorizeDevice(ctx context.Context, c *client, show func(DeviceAuthorization) error) (string, error) {
	if c.params.ClientID == "" {
		return "", errors.New("simkl: client_id is required")
	}
	f := c.form()
	f.Set("scope", "media:read")
	var d DeviceAuthorization
	if err := c.post(ctx, "/oauth2/device", f, &d); err != nil {
		return "", err
	}
	if d.DeviceCode == "" || d.UserCode == "" || d.VerificationURI == "" || d.ExpiresIn <= 0 || d.ExpiresIn > 900 || d.Interval < 0 || d.Interval > 900 {
		return "", errors.New("simkl: invalid device authorization response")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(d.ExpiresIn)*time.Second)
	defer cancel()
	if show == nil {
		return "", errors.New("simkl: authorization instructions callback required")
	}
	if err := show(d); err != nil {
		return "", err
	}
	interval := time.Duration(d.Interval) * time.Second
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	deadline := c.now().Add(time.Duration(d.ExpiresIn) * time.Second)
	for c.now().Before(deadline) {
		if err := c.sleep(ctx, interval); err != nil {
			return "", err
		}
		if !c.now().Before(deadline) {
			break
		}
		f = c.form()
		f.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
		f.Set("device_code", d.DeviceCode)
		var t tokenResponse
		err := c.post(ctx, "/oauth2/token", f, &t)
		var ae *APIError
		if errors.As(err, &ae) {
			if ae.Code == "authorization_pending" {
				continue
			}
			if ae.Code == "slow_down" {
				interval += 5 * time.Second
				continue
			}
		}
		if err != nil {
			return "", err
		}
		if err = validToken(t); err != nil {
			return "", err
		}
		// Each client starts with no access token and only the configured refresh token.
		for range 2 {
			fresh := newClient(Params{ClientID: c.params.ClientID, ClientSecret: c.params.ClientSecret, RefreshToken: t.RefreshToken})
			fresh.http = c.http
			fresh.api = c.api
			fresh.now = c.now
			fresh.sleep = c.sleep
			fresh.pace = c.pace
			fresh.gate = c.gate
			fresh.postNext = c.postNext
			if _, err = fresh.accessToken(ctx, ""); err != nil {
				return "", err
			}
			c.postNext = fresh.postNext
		}
		return t.RefreshToken, nil
	}
	return "", errors.New("simkl: device authorization expired; run setup again")
}
