package server

import (
	"net/http"

	"github.com/gorilla/mux"
	polochon "github.com/odwrtw/polochon/lib"
)

func getLanguage(r *http.Request) polochon.Language {
	vars := mux.Vars(r)
	return polochon.Language(vars["lang"])
}

func (s *Server) updateMovieSubtitle(w http.ResponseWriter, r *http.Request) {
	s.logEntry(r).Infof("updating movie subtitles")

	m := s.getMovie(w, r)
	if m == nil {
		return
	}

	s.updateSubtitle(m, w, r)
}

func (s *Server) updateEpisodeSubtitle(w http.ResponseWriter, r *http.Request) {
	s.logEntry(r).Infof("updating episode subtitles")

	e := s.getEpisode(w, r)
	if e == nil {
		return
	}

	s.updateSubtitle(e, w, r)
}

func (s *Server) updateSubtitle(v polochon.Video, w http.ResponseWriter, r *http.Request) {
	if v == nil {
		return
	}
	v.SetSubtitles(nil)

	log := s.logEntry(r)

	sub, err := polochon.GetSubtitle(v, getLanguage(r), log)
	if err != nil {
		if err != polochon.ErrNoSubtitleFound {
			s.renderError(w, r, err)
		}
		return
	}

	// Save in the library
	if err := s.library.SaveSubtitles(v, log); err != nil {
		s.renderError(w, r, err)
		return
	}

	// Save in the media index
	if err := s.library.UpdateSubtitleIndex(v, sub); err != nil {
		s.renderError(w, r, err)
		return
	}

	s.renderOK(w, sub)
}

func (s *Server) serveSubtitle(v polochon.Video, w http.ResponseWriter, r *http.Request) {
	if v == nil {
		return
	}

	sub := polochon.NewSubtitleFromVideo(v, getLanguage(r))
	s.serveFile(w, r, &sub.File)
}

func (s *Server) serveMovieSubtitle(w http.ResponseWriter, req *http.Request) {
	m := s.getMovie(w, req)
	if m == nil {
		return
	}
	s.serveSubtitle(m, w, req)
}

func (s *Server) serveEpisodeSubtitle(w http.ResponseWriter, req *http.Request) {
	e := s.getEpisode(w, req)
	if e == nil {
		return
	}

	s.serveSubtitle(e, w, req)
}
