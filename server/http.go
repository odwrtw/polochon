package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gitlab.quimbo.fr/odwrtw/polochon/lib"
)

// hello world, the web server
func (a *App) movieStore(w http.ResponseWriter, req *http.Request) {
	vs := polochon.NewVideoStore(a.config)

	movies, err := vs.ScanMovies()
	if err != nil {
		a.writeError(w, err.Error())
		return
	}

	a.writeResponse(w, movies)
}

func (a *App) showStore(w http.ResponseWriter, req *http.Request) {
	vs := polochon.NewVideoStore(a.config)

	shows, err := vs.ScanShows()
	if err != nil {
		a.writeError(w, err.Error())
		return
	}

	a.writeResponse(w, shows)
}

// HTTPServer handles the HTTP requests
func (a *App) HTTPServer() {
	addr := fmt.Sprintf("%s:%d", a.config.HTTPServer.Host, a.config.HTTPServer.Port)
	s := &http.Server{
		Addr: addr,
	}
	http.HandleFunc("/videos/movies", a.movieStore)
	http.HandleFunc("/videos/shows", a.showStore)

	// Serve HTTP
	if err := s.ListenAndServe(); err != nil {
		a.config.Log.Error("Couldn't start the HTTP server : ", err)
		a.Stop()
	}
}

// writeResponse helps write a json formatted response into the ResponseWriter
func (a *App) writeResponse(w http.ResponseWriter, v interface{}) {
	b, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		msg := fmt.Sprintf("Failed to encode json response: %+v", v)
		a.writeError(w, msg)
		return
	}

	w.Write(b)
}

func (a *App) writeError(w http.ResponseWriter, msg string) {
	a.config.Log.Errorf(msg)
	http.Error(w, msg, http.StatusInternalServerError)
}
