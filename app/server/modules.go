package server

import "net/http"

func (s *Server) getModulesStatus(w http.ResponseWriter, req *http.Request) {
	status := s.config.ModulesStatus()

	s.renderOK(w, status)
}
