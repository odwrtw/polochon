package token

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
	token := r.URL.Query().Get("token")

	var match mux.RouteMatch

	// Unknow route let the app handle that case ie 404
	if ok := m.router.Match(r, &match); !ok {
		next(w, r)
		return
	}

	// No route name forbiden
	if match.Route.GetName() == "" {
		http.Error(w, "Invalid route", http.StatusForbidden)
		return
	}

	routeName := match.Route.GetName()

	if ok := m.manager.IsAllowed(token, routeName); !ok {
		http.Error(w, "Invalid token", http.StatusForbidden)
		return
	}

	next(w, r)
}
