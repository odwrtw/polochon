package server

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/odwrtw/polochon/lib"
)

func (s *Server) movieIds(w http.ResponseWriter, req *http.Request) {
	s.log.Debug("listing movies by ids")
	s.renderOK(w, s.library.MovieIDs())
}

func (s *Server) movieIndex(w http.ResponseWriter, req *http.Request) {
	s.log.Debug("listing movie index")
	s.renderOK(w, s.library.MovieIndex())
}

// TODO: handle this in a middleware
func (s *Server) getMovie(w http.ResponseWriter, req *http.Request) *polochon.Movie {
	vars := mux.Vars(req)
	id := vars["id"]

	s.log.Debugf("looking for a movie with ID %q", id)

	// Find the file
	m, err := s.library.GetMovie(id)
	if err != nil {
		s.renderError(w, err)
		return nil
	}

	return m
}

func (s *Server) getMovieDetails(w http.ResponseWriter, req *http.Request) {
	m := s.getMovie(w, req)
	if m == nil {
		return
	}

	idxMovie, err := s.library.GetIndexedMovie(m.ImdbID)
	if err != nil {
		s.renderError(w, err)
		return
	}

	movie := struct {
		*polochon.Movie
		Subtitles []polochon.Language `json:"subtitles"`
	}{
		Movie:     m,
		Subtitles: idxMovie.Subtitles,
	}

	s.renderOK(w, movie)
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
		s.renderError(w, err)
		return
	}

	s.renderOK(w, nil)
}
