package server

import (
	"net/http"
)

func (s *Server) libraryRefresh(w http.ResponseWriter, req *http.Request) {
	log := s.logEntry(req)
	log.Info("refreshing library")

	if err := s.library.RebuildIndex(); err != nil {
		log.With("function", "rebuild_index").Error(err.Error())
		s.renderError(w, req, err)
		return
	}

	s.hub.broadcast()
	s.renderOK(w, nil)
}
