package server

import (
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/meatballhat/negroni-logrus"
	"github.com/odwrtw/polochon/app/internal/token"
	"github.com/phyber/negroni-gzip/gzip"
)

// httpServer returns an http server
func (s *Server) httpServer(log *logrus.Entry) *http.Server {
	addr := fmt.Sprintf("%s:%d", s.config.HTTPServer.Host, s.config.HTTPServer.Port)
	log.Debugf("http server will listen on: %s", addr)

	mux := mux.NewRouter()

	mux.HandleFunc("/movies", s.movieIds).Name("MoviesListIDs").Methods("GET")
	mux.HandleFunc("/movies/{id}", s.getMovieDetails).Name("GetMovieDetails").Methods("GET")
	mux.HandleFunc("/movies/{id}", s.deleteMovie).Name("DeleteMovie").Methods("DELETE")

	mux.HandleFunc("/shows", s.showIds).Name("ShowsListIDs").Methods("GET")
	mux.HandleFunc("/shows/{id}", s.getShowDetails).Name("GetShowDetails").Methods("GET")
	mux.HandleFunc("/shows/{id}/{season:[0-9]+}/{episode:[0-9]+}", s.getShowEpisodeIDDetails).Name("GetShowEpisodeIDDetails").Methods("GET")
	mux.HandleFunc("/shows/{id}/{season:[0-9]+}/{episode:[0-9]+}", s.deleteEpisode).Name("DeleteEpisode").Methods("DELETE")
	mux.HandleFunc("/wishlist", s.wishlist).Name("Wishlist").Methods("GET")

	mux.HandleFunc("/torrents", s.addTorrent).Name("TorrentsAdd").Methods("POST")

	if s.config.HTTPServer.ServeFiles {
		log.Debug("server will be serving files")
		mux.HandleFunc("/shows/{id}/{season:[0-9]+}/{episode:[0-9]+}/download", s.serveShow).Name("ServeShowsByIDs").Methods("GET")
		mux.HandleFunc("/movies/{id}/download", s.serveMovie).Name("ServeMoviesByIDs").Methods("GET")
	}

	n := negroni.New()

	// Panic recovery
	n.Use(negroni.NewRecovery())

	// Use logrus as logger
	n.Use(negronilogrus.NewMiddlewareFromLogger(s.log.Logger, "httpServer"))

	// gzip compression
	n.Use(gzip.Gzip(gzip.DefaultCompression))

	// Add basic auth if configured
	if s.config.HTTPServer.BasicAuth {
		log.Info("server will require basic authentication")
		n.Use(NewBasicAuthMiddleware(s.config.HTTPServer.BasicAuthUser, s.config.HTTPServer.BasicAuthPassword))
	}

	// Add token auth middleware if token configuration file specified
	if s.tokenManager != nil {
		n.Use(token.NewMiddleware(s.tokenManager, mux))
		mux.HandleFunc("/tokens/allowed", s.tokenGetAllowed).Name("TokenGetAllowed")
	}

	// Wrap the router
	n.UseHandler(mux)

	return &http.Server{Addr: addr, Handler: n}
}
