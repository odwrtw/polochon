package auth

import (
	"net/http"

	"github.com/gorilla/mux"
)

// Middleware used for check the token and access rigth
type Middleware struct {
	manager *Manager
	router  *mux.Router
}

// NewMiddleware returns a new token middleware
func NewMiddleware(manager *Manager, router *mux.Router) *Middleware {
	return &Middleware{
		manager: manager,
		router:  router,
	}
}

// ServeHTTP implements the negroni middleware interface
func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	token := r.Header.Get("X-Auth-Token")
	if token == "" {
		token = r.URL.Query().Get("token")
	}

	var match mux.RouteMatch

	if !m.router.Match(r, &match) {
		http.NotFound(w, r)
		return
	}

	// Get the route name
	routeName := match.Route.GetName()
	if routeName == "" {
		http.NotFound(w, r)
		return
	}

	if !m.manager.IsAllowed(token, routeName) {
		http.NotFound(w, r)
		return
	}

	next(w, r)
}
