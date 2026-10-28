package simkl

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestDeviceFlowAndFixedTokenVerification(t *testing.T) {
	var polls, refreshes atomic.Int32
	s := testModule(t, Params{ClientID: "client", ClientSecret: "client-secret"}, func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("client_id") != "client" || r.Form.Get("client_secret") != "client-secret" {
			t.Error("missing registration credentials")
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("bearer token on OAuth request")
		}
		if r.URL.Path == "/oauth2/device" {
			if r.Form.Get("scope") != "media:read" {
				t.Error("requested write scope")
			}
			writeJSON(w, DeviceAuthorization{DeviceCode: "private-device-code", UserCode: "ABCD-EFGH", VerificationURI: "https://simkl.com/pin", ExpiresIn: 900, Interval: 5})
			return
		}
		if r.Form.Get("grant_type") == "refresh_token" {
			refreshes.Add(1)
			if r.Form.Get("refresh_token") != "refresh-secret" {
				t.Error("configured token changed")
			}
			tokenJSON(w, "refreshed-access")
			return
		}
		if r.Form.Get("device_code") != "private-device-code" {
			t.Error("missing device code")
		}
		switch polls.Add(1) {
		case 1:
			w.WriteHeader(400)
			writeJSON(w, map[string]string{"error": "authorization_pending"})
		case 2:
			w.WriteHeader(400)
			writeJSON(w, map[string]string{"error": "slow_down"})
		default:
			tokenJSON(w, "initial-access")
		}
	})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s.client.now = func() time.Time { return now }
	var waits []time.Duration
	s.client.sleep = func(ctx context.Context, d time.Duration) error {
		if d > 0 {
			waits = append(waits, d)
			now = now.Add(d)
		}
		return ctx.Err()
	}
	shown := false
	token, err := authorizeDevice(context.Background(), s.client, func(d DeviceAuthorization) error { shown = d.UserCode == "ABCD-EFGH"; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if !shown || token != "refresh-secret" || polls.Load() != 3 || refreshes.Load() != 2 {
		t.Fatalf("incomplete setup verification: %s %d %d", token, polls.Load(), refreshes.Load())
	}
	if len(waits) < 3 || waits[0] != 5*time.Second || waits[1] != 5*time.Second || waits[2] != 10*time.Second {
		t.Fatalf("incorrect slow_down waits: %v", waits)
	}
}

func TestDeviceExpiryCancellationAndScope(t *testing.T) {
	for _, mode := range []string{"expiry", "cancel", "scope", "rotation", "invalid-client"} {
		t.Run(mode, func(t *testing.T) {
			s := testModule(t, Params{ClientID: "client"}, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/oauth2/device" {
					writeJSON(w, DeviceAuthorization{DeviceCode: "code", UserCode: "ABCD-EFGH", VerificationURI: "https://simkl.com/pin", ExpiresIn: 10, Interval: 5})
					return
				}
				if mode == "invalid-client" {
					w.WriteHeader(401)
					writeJSON(w, map[string]string{"error": "invalid_client"})
					return
				}
				if mode == "scope" {
					writeJSON(w, map[string]any{"access_token": "access", "refresh_token": "refresh-secret", "expires_in": 600, "token_type": "Bearer", "scope": "unrecognized"})
					return
				}
				if mode == "rotation" {
					_ = r.ParseForm()
					if r.Form.Get("grant_type") == "refresh_token" {
						writeJSON(w, map[string]any{"access_token": "access", "refresh_token": "different-secret", "expires_in": 600, "token_type": "Bearer", "scope": "media:read"})
						return
					}
					tokenJSON(w, "access")
					return
				}
				w.WriteHeader(400)
				writeJSON(w, map[string]string{"error": "authorization_pending"})
			})
			now := time.Now()
			s.client.now = func() time.Time { return now }
			s.client.sleep = func(ctx context.Context, d time.Duration) error {
				if d > 0 {
					now = now.Add(d)
				}
				return ctx.Err()
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, err := authorizeDevice(ctx, s.client, func(DeviceAuthorization) error {
				if mode == "cancel" {
					cancel()
				}
				return nil
			})
			if err == nil {
				t.Fatal("expected setup failure")
			}
			if mode == "scope" && !errors.Is(err, ErrScope) {
				t.Fatal(err)
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
}
