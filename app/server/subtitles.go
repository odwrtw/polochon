package server

import (
	"net/http"

	"github.com/gorilla/mux"
	polochon "github.com/odwrtw/polochon/lib"
)

func getLanguage(w http.ResponseWriter, req *http.Request) polochon.Language {
	vars := mux.Vars(req)
	lang := vars["lang"]

	return polochon.Language(lang)
}

func (s *Server) updateMovieSubtitles(w http.ResponseWriter, req *http.Request) {
	log := s.logEntry(req)
	log.Infof("updating movie subtitles")

	m := s.getMovie(w, req)
	if m == nil {
		return
	}

	subs, err := s.library.AddSubtitles(m, s.config.SubtitleLanguages, log)
	if err != nil {
		s.renderError(w, req, err)
		return
	}

	s.renderOK(w, subs)
}

func (s *Server) updateEpisodeSubtitles(w http.ResponseWriter, req *http.Request) {
	log := s.logEntry(req)
	log.Infof("updating episode subtitles")

	e := s.getEpisode(w, req)
	if e == nil {
		return
	}

	subs, err := s.library.AddSubtitles(e, s.config.SubtitleLanguages, log)
	if err != nil {
		s.renderError(w, req, err)
		return
	}

	s.renderOK(w, subs)
}

func (s *Server) serveMovieSubtitle(w http.ResponseWriter, req *http.Request) {
	m := s.getMovie(w, req)
	if m == nil {
		return
	}

	lang := getLanguage(w, req)
	path := m.SubtitlePath(polochon.Language(lang))

	file := &polochon.File{
		Path: path,
	}

	s.serveFile(w, req, file)
}

func (s *Server) serveEpisodeSubtitle(w http.ResponseWriter, req *http.Request) {
	e := s.getEpisode(w, req)
	if e == nil {
		return
	}

	lang := getLanguage(w, req)
	path := e.SubtitlePath(polochon.Language(lang))

	file := &polochon.File{
		Path: path,
	}

	s.serveFile(w, req, file)
}
