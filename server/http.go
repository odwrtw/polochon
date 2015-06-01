package main

import (
	"bufio"
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

func (a *App) serveFiles(w http.ResponseWriter, req *http.Request) {
	var fileServer http.Handler

	user, pwd, ok := req.BasicAuth()
	if ok != true {
		http.Error(w, "401 unauthorized", http.StatusUnauthorized)
		return
	}
	if user != a.config.HTTPServer.ServeFilesUser || pwd != a.config.HTTPServer.ServeFilesPwd {
		http.Error(w, "401 unauthorized", http.StatusUnauthorized)
		return
	}

	vtype := req.FormValue("type")
	if vtype != "movie" && vtype != "show" {
		http.Error(w, "400 bad request", http.StatusBadRequest)
		return
	}
	rfile := req.FormValue("file")
	if rfile == "" {
		http.Error(w, "400 bad request", http.StatusBadRequest)
		return
	}

	if vtype == "movie" {
		fileServer = http.FileServer(http.Dir(a.config.Movie.Dir))
	}
	if vtype == "show" {
		fileServer = http.FileServer(http.Dir(a.config.Show.Dir))
	}

	freq, err := http.NewRequest("GET", rfile, &bufio.Reader{})
	if err != nil {
		http.Error(w, "500 internal server error", http.StatusInternalServerError)
	}

	fileServer.ServeHTTP(w, freq)
}

// HTTPServer handles the HTTP requests
func (a *App) HTTPServer() {
	addr := fmt.Sprintf("%s:%d", a.config.HTTPServer.Host, a.config.HTTPServer.Port)
	s := &http.Server{
		Addr: addr,
	}
	http.HandleFunc("/videos/movies", a.movieStore)
	http.HandleFunc("/videos/shows", a.showStore)

	if a.config.HTTPServer.ServeFiles {
		http.HandleFunc("/files", a.serveFiles)
	}

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
