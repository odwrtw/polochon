package server

import (
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	polochon "github.com/odwrtw/polochon/lib"
	index "github.com/odwrtw/polochon/lib/media_index"
)

func (s *Server) movieIndex(w http.ResponseWriter, req *http.Request) {
	s.logEntry(req).Infof("listing movie index")
	s.renderOK(w, s.library.MovieIndex())
}

// TODO: handle this in a middleware
func (s *Server) getMovie(w http.ResponseWriter, req *http.Request) *polochon.Movie {
	vars := mux.Vars(req)
	id := vars["id"]

	s.logEntry(req).Infof("looking for a movie with ID %q", id)

	// Find the file
	m, err := s.library.GetMovie(id)
	if err != nil {
		s.renderError(w, req, err)
		return nil
	}

	return m
}

func (s *Server) getMovieDetails(w http.ResponseWriter, req *http.Request) {
	s.logEntry(req).Infof("getting movie details")

	m := s.getMovie(w, req)
	if m == nil {
		return
	}

	idxMovie, err := s.library.GetIndexedMovie(m.ImdbID)
	if err != nil {
		s.renderError(w, req, err)
		return
	}

	s.renderOK(w, idxMovie)
}

func (s *Server) serveMovie(w http.ResponseWriter, req *http.Request) {
	m := s.getMovie(w, req)
	if m == nil {
		return
	}

	s.serveFile(w, req, m.GetFile())
}

func (s *Server) serveMovieFile(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	name := vars["name"]

	movie, err := s.library.GetMovie(id)
	if err != nil {
		s.renderError(w, req, err)
		return
	}

	var path string
	for _, p := range []string{
		movie.NfoPath(),
		movie.MovieThumbPath(),
		movie.MovieFanartPath(),
	} {
		if name == filepath.Base(p) {
			path = p
			break
		}
	}

	if path == "" {
		s.renderError(w, req, index.ErrNotFound)
		return
	}

	s.serveFile(w, req, polochon.NewFile(path))
}

func (s *Server) deleteMovie(w http.ResponseWriter, req *http.Request) {
	log := s.logEntry(req)
	log.Infof("deleting movie")

	m := s.getMovie(w, req)
	if m == nil {
		return
	}

	if err := s.library.Delete(m, log); err != nil {
		s.renderError(w, req, err)
		return
	}

	s.renderOK(w, nil)
}
