package main

import (
	"fmt"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/meatballhat/negroni-logrus"
	"github.com/odwrtw/polochon/lib"
)

// hello world, the web server
func (a *App) movieStore(w http.ResponseWriter, req *http.Request) {
	vs := polochon.NewVideoStore(a.config, a.logger)

	movies, err := vs.ScanMovies()
	if err != nil {
		a.render.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	toJSONMovies := []polochon.Movie{}
	for _, m := range movies {
		movie, err := m.PrepareForJSON()
		if err != nil {
			msg := fmt.Sprintf("Failed to prepare for json response: %+v", err)
			a.render.JSON(w, http.StatusInternalServerError, map[string]string{"error": msg})
		}
		toJSONMovies = append(toJSONMovies, movie)
	}
	a.render.JSON(w, http.StatusOK, toJSONMovies)
}

func (a *App) showStore(w http.ResponseWriter, req *http.Request) {
	vs := polochon.NewVideoStore(a.config, a.logger)

	shows, err := vs.ScanShows()
	if err != nil {
		a.render.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	toJSONShows := []polochon.Show{}
	for _, s := range shows {
		show, err := s.PrepareForJSON()
		if err != nil {
			msg := fmt.Sprintf("Failed to prepare fo json response: %+v", err)
			a.render.JSON(w, http.StatusInternalServerError, map[string]string{"error": msg})
		}
		toJSONShows = append(toJSONShows, show)
	}

	a.render.JSON(w, http.StatusOK, toJSONShows)
}

func (a *App) serveFiles(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	slug := vars["slug"]

	// TODO: create the real file handler with video slugs
	a.logger.Infof("Should get file with slug: %s", slug)

	a.render.JSON(w, http.StatusInternalServerError, map[string]string{"slug": slug})
}

// HTTPServer handles the HTTP requests
func (a *App) HTTPServer() {
	addr := fmt.Sprintf("%s:%d", a.config.HTTPServer.Host, a.config.HTTPServer.Port)

	a.mux.HandleFunc("/videos/movies", a.movieStore)
	a.mux.HandleFunc("/videos/shows", a.showStore)

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
