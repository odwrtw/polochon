package server

import (
	"net/http"
)

func (s *Server) libraryRefresh(w http.ResponseWriter, req *http.Request) {
	log := s.logEntry(req)
	log.Infof("refreshing library")

	if err := s.library.RebuildIndex(log); err != nil {
		log.WithField("function", "rebuild_index").Error(err)
		s.renderError(w, req, err)
	}

	s.renderOK(w, nil)
}
