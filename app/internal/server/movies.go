package server

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/media_index"
)

func (s *Server) movieIds(w http.ResponseWriter, req *http.Request) {
	s.log.Debug("listing movies by ids")

	movieIds, err := s.library.MovieIDs()
	if err != nil {
		s.renderError(w, err)
		return
	}
	s.renderOK(w, movieIds)
}

// TODO: handle this in a middleware
func (s *Server) getMovie(w http.ResponseWriter, req *http.Request) *polochon.Movie {
	vars := mux.Vars(req)
	id := vars["id"]

	s.log.Debugf("looking for a movie with ID %q", id)

	// Find the file
	m, err := s.library.GetMovie(id)
	if err != nil {
		s.log.Error(err)
		var status int
		if err == index.ErrNotFound {
			status = http.StatusNotFound
		} else {
			status = http.StatusInternalServerError
		}
		s.renderError(w, &Error{
			Code:    status,
			Message: "URL not found",
		})
		return nil
	}

	return m
}

func (s *Server) getMovieDetails(w http.ResponseWriter, req *http.Request) {
	m := s.getMovie(w, req)
	if m == nil {
		return
	}

	s.renderOK(w, m)
}

func (s *Server) serveMovie(w http.ResponseWriter, req *http.Request) {
	m := s.getMovie(w, req)
	if m == nil {
		return
	}

	s.serveFile(w, req, m.GetFile())
}

func (s *Server) deleteMovie(w http.ResponseWriter, req *http.Request) {
	m := s.getMovie(w, req)
	if m == nil {
		return
	}

	if err := s.library.Delete(m, s.log); err != nil {
		s.renderError(w, &Error{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	s.renderOK(w, nil)
}
