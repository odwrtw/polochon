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

	mux.HandleFunc("/movies/slugs", s.movieSlugs).Name("MoviesListSlugs")
	mux.HandleFunc("/movies/ids", s.movieIds).Name("MoviesListIDs")
	mux.HandleFunc("/movies/{idType:ids|slugs}/{id}", s.getMovieDetails).Name("GetMovieDetails")

	mux.HandleFunc("/shows/ids", s.showIds).Name("ShowsListIDs")
	mux.HandleFunc("/shows/slugs", s.showSlugs).Name("ShowsListSlugs")
	mux.HandleFunc("/wishlist", s.wishlist).Name("Wishlist")

	mux.HandleFunc("/torrents", s.addTorrent).Methods("POST").Name("TorrentsAdd")

	if s.config.HTTPServer.ServeFiles {
		log.Debug("server will be serving files")
		mux.HandleFunc("/{videoType:movies|shows}/slugs/{slug}/delete", s.deleteFile).Name("DeleteBySlugs")
		mux.HandleFunc("/shows/slugs/{slug}/download", s.serveShow).Name("ServeShowsBySlugs")
		mux.HandleFunc("/shows/ids/{id}/{season}/{episode}/download", s.serveShow).Name("ServeShowsByIDs")
		mux.HandleFunc("/movies/ids/{id}/download", s.serveMovie).Name("ServeMoviesByIDs")
		mux.HandleFunc("/movies/slugs/{slug}/download", s.serveMovie).Name("ServeMoviesBySlugs")
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
