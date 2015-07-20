package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/meatballhat/negroni-logrus"
	"github.com/odwrtw/polochon/lib"
)

// hello world, the web server
func (a *App) movieStore(w http.ResponseWriter, req *http.Request) {
	a.logger.Debug("Listing movies")
	vs := polochon.NewVideoStore(a.config, a.logger)

	movies, err := vs.ScanMovies()
	if err != nil {
		a.render.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	a.render.JSON(w, http.StatusOK, movies)
}

func (a *App) showStore(w http.ResponseWriter, req *http.Request) {
	a.logger.Debug("Listing shows")
	vs := polochon.NewVideoStore(a.config, a.logger)

	shows, err := vs.ScanShows()
	if err != nil {
		a.render.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	a.render.JSON(w, http.StatusOK, shows)
}

func (a *App) serveFiles(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	slug := vars["slug"]
	slug = strings.ToLower(slug)

	a.logger.Debugf("Looking for: %s", slug)

	// Instanciate a VideoStore to look for the Slug
	vs := polochon.NewVideoStore(a.config, a.logger)

	// Find the file by Slug
	f, err := vs.SearchFileBySlug(slug)
	if err != nil {
		a.logger.Error(err.Error())
		var status int
		if err == polochon.ErrSlugNotFound {
			status = http.StatusNotFound
		} else {
			status = http.StatusInternalServerError
		}
		a.render.JSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	a.logger.Debugf("Going to serve: %s", filepath.Base(f.Path))

	// Set the header so that when downloading, the real filename will be given
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(f.Path)))
	http.ServeFile(w, req, f.Path)
}

// HTTPServer handles the HTTP requests
func (a *App) HTTPServer() {
	addr := fmt.Sprintf("%s:%d", a.config.HTTPServer.Host, a.config.HTTPServer.Port)
	a.logger.Debugf("HTTP Server listening on: %s", addr)

	// s will handle /videos requests
	s := a.mux.PathPrefix("/videos").Subrouter()
	// /videos/shows
	s.HandleFunc("/shows", a.showStore)
	// /videos/movies
	s.HandleFunc("/movies", a.movieStore)

	if a.config.HTTPServer.ServeFiles {
		a.logger.Info("Server is serving files")
		a.mux.HandleFunc("/files/{slug}", a.serveFiles)
	}

	n := negroni.New()
	// Panic recovery
	n.Use(negroni.NewRecovery())
	// Use logrus as logger
	n.Use(negronilogrus.NewCustomMiddleware(logrus.InfoLevel, a.logger.Formatter, "httpServer"))

	// Add basic auth if configured
	if a.config.HTTPServer.BasicAuth {
		a.logger.Info("Server requires basic authentication")
		n.Use(NewBasicAuthMiddleware(a.config.HTTPServer.BasicAuthUser, a.config.HTTPServer.BasicAuthPassword))
	}

	// Wrap the router
	n.UseHandler(a.mux)

	// Serve HTTP
	if err := http.ListenAndServe(addr, n); err != nil {
		a.logger.Error("Couldn't start the HTTP server : ", err)
		a.Stop()
	}
}
