package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/urfave/negroni"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	m, err := New(strings.NewReader(`
- role: reader
  read: true
  write: false
  debug: false
  token:
  - name: user token 1
    value: token1
`))
	if err != nil {
		t.Fatalf("failed to create manager: %s", err)
	}
	return m
}

func TestMiddleWare(t *testing.T) {
	manager := newTestManager(t)

	mux := http.NewServeMux()
	mux.HandleFunc("/stuff", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/private/write", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	n := negroni.New()
	n.Use(NewMiddleware(manager))
	n.UseHandler(mux)

	server := httptest.NewServer(n)
	defer server.Close()

	tt := []struct {
		name           string
		path           string
		method         string
		token          string
		useHeader      bool
		expectedStatus int
	}{
		{
			name: "valid token in url param",
			path: "/stuff", method: "GET",
			token: "token1", useHeader: false,
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid token in header",
			path: "/stuff", method: "GET",
			token: "token1", useHeader: true,
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid token",
			path: "/stuff", method: "GET",
			token:          "badtoken",
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "reader cannot write",
			path: "/private/write", method: "POST",
			token:          "token1",
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "reader cannot debug",
			path: "/debug/pprof/", method: "GET",
			token:          "token1",
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "reader cannot access metrics",
			path: "/metrics", method: "GET",
			token:          "token1",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			url := server.URL + tc.path
			if !tc.useHeader && tc.token != "" {
				url += "?token=" + tc.token
			}

			req, err := http.NewRequest(tc.method, url, nil)
			if err != nil {
				t.Fatal(err)
			}
			if tc.useHeader {
				req.Header.Add("X-Auth-Token", tc.token)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}

			if resp.StatusCode != tc.expectedStatus {
				t.Fatalf("want %d, got %d", tc.expectedStatus, resp.StatusCode)
			}
		})
	}
}

func TestMiddlewareCtx(t *testing.T) {
	manager := newTestManager(t)
	middleware := NewMiddleware(manager)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/stuff", nil)
	if err != nil {
		t.Fatalf("creating request: %s", err)
	}
	r.Header.Add("X-Auth-Token", "token1")

	middleware.ServeHTTP(w, r, func(w http.ResponseWriter, r *http.Request) {
		tokenName, ok := r.Context().Value(TokenName).(string)
		if !ok {
			t.Fatal("expected token name in context")
		}
		if tokenName != "user token 1" {
			t.Fatalf("invalid token name: %q", tokenName)
		}
	})
}
