package server

import (
	"context"
	"net/http"
	"strings"
)

type authCtxKey string

const tokenName authCtxKey = "auth-token-name"

type authMiddleware struct {
	manager *authManager
}

func newAuthMiddleware(manager *authManager) *authMiddleware {
	return &authMiddleware{manager: manager}
}

func rightForRequest(r *http.Request) authRight {
	if strings.HasPrefix(r.URL.Path, "/debug/") || r.URL.Path == "/metrics" {
		return authRightDebug
	}
	if r.Method == http.MethodGet {
		return authRightRead
	}
	return authRightWrite
}

// ServeHTTP implements the negroni middleware interface.
func (m *authMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	token := r.Header.Get("X-Auth-Token")
	if token == "" {
		token = r.URL.Query().Get("token")
	}

	name, ok := m.manager.isAllowed(token, rightForRequest(r))
	if !ok {
		http.NotFound(w, r)
		return
	}

	ctx := context.WithValue(r.Context(), tokenName, name)
	next(w, r.WithContext(ctx))
}
