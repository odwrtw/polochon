package auth

import (
	"context"
	"net/http"
	"strings"
)

// CtxKey is a type of context key.
type CtxKey string

// TokenName is the key used in the context to store the token's display name.
const TokenName CtxKey = "auth-token-name"

// Middleware checks token rights for each request.
type Middleware struct {
	manager *Manager
}

// NewMiddleware returns a new token middleware.
func NewMiddleware(manager *Manager) *Middleware {
	return &Middleware{manager: manager}
}

// rightForRequest determines which right a request requires.
func rightForRequest(r *http.Request) Right {
	if strings.HasPrefix(r.URL.Path, "/debug/") || r.URL.Path == "/metrics" {
		return RightDebug
	}
	if r.Method == http.MethodGet {
		return RightRead
	}
	return RightWrite
}

// ServeHTTP implements the negroni middleware interface.
func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	token := r.Header.Get("X-Auth-Token")
	if token == "" {
		token = r.URL.Query().Get("token")
	}

	name, ok := m.manager.IsAllowed(token, rightForRequest(r))
	if !ok {
		http.NotFound(w, r)
		return
	}

	ctx := context.WithValue(r.Context(), TokenName, name)
	next(w, r.WithContext(ctx))
}
