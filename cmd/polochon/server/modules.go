package server

import "net/http"

func (s *Server) getModulesStatus(w http.ResponseWriter, r *http.Request) {
	s.logEntry(r).Info("getting modules status")
	status := s.config.ModulesStatus()

	s.renderOK(w, status)
}
