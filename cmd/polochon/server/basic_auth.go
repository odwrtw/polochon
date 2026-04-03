package server

import "net/http"

// BasicAuthMiddleware holds the informations to build a basic authentication
// middleware
type BasicAuthMiddleware struct {
	Username string
	Password string
}

// NewBasicAuthMiddleware returns a new basic auth middleware
func NewBasicAuthMiddleware(username, password string) *BasicAuthMiddleware {
	return &BasicAuthMiddleware{
		Username: username,
		Password: password,
	}
}

// ServeHTTP implements the negroni middleware interface
func (ba *BasicAuthMiddleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	user, pwd, ok := r.BasicAuth()
	if !ok || user != ba.Username || pwd != ba.Password {
		rw.Header().Set("WWW-Authenticate", `Basic realm="User Auth"`)
		http.Error(rw, "401 unauthorized", http.StatusUnauthorized)
		return
	}

	next(rw, r)
}
