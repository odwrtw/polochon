package main

import (
	"fmt"
	"net/http"
	"path/filepath"

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

func (a *App) serveFile(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	videoType := vars["videoType"]
	slug := vars["slug"]

	a.logger.Debugf("Looking for the %s: %s", videoType, slug)

	// Instanciate a VideoStore to look for the Slug
	vs := polochon.NewVideoStore(a.config, a.logger)

	var searchFunc func(slug string) (polochon.Video, error)
	switch videoType {
	case "movies":
		searchFunc = vs.SearchMovieBySlug
	case "shows":
		searchFunc = vs.SearchShowEpisodeBySlug
	default:
		msg := fmt.Sprintf("Invalid video type: %q", videoType)
		a.render.JSON(w, http.StatusInternalServerError, map[string]string{"error": msg})
		return
	}

	// Find the file by Slug
	v, err := searchFunc(slug)
	if err != nil {
		a.logger.Error(err)
		var status int
		if err == polochon.ErrSlugNotFound {
			status = http.StatusNotFound
		} else {
			status = http.StatusInternalServerError
		}
		a.render.JSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	videoFile := v.GetFile()
	a.logger.Debugf("Going to serve: %s", filepath.Base(videoFile.Path))

	// Set the header so that when downloading, the real filename will be given
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(videoFile.Path)))
	http.ServeFile(w, req, videoFile.Path)
}

// HTTPServer handles the HTTP requests
func (a *App) HTTPServer() {
	addr := fmt.Sprintf("%s:%d", a.config.HTTPServer.Host, a.config.HTTPServer.Port)
	a.logger.Debugf("HTTP Server listening on: %s", addr)

	// /videos/shows
	a.mux.HandleFunc("/shows", a.showStore)
	// /videos/movies
	a.mux.HandleFunc("/movies", a.movieStore)

	if a.config.HTTPServer.ServeFiles {
		a.logger.Info("Server is serving files")
		a.mux.HandleFunc("/{videoType:movies|shows}/{slug}/download", a.serveFile)
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
