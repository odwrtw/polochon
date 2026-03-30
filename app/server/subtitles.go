package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	polochon "github.com/odwrtw/polochon/lib"
	index "github.com/odwrtw/polochon/lib/media_index"
)

func (s *Server) listSubtitles(v polochon.Video, w http.ResponseWriter, r *http.Request) {
	log := s.logEntry(r)

	lang, err := getLanguage(r)
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	entries, err := polochon.ListSubtitles(v, lang, log)
	if err != nil && err != polochon.ErrNoSubtitleFound {
		s.renderError(w, r, err)
		return
	}

	s.renderOK(w, entries)
}

func (s *Server) downloadSubtitleByEntry(v polochon.Video, w http.ResponseWriter, r *http.Request) {
	log := s.logEntry(r)

	lang, err := getLanguage(r)
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	var entry polochon.SubtitleEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		s.renderError(w, r, err)
		return
	}
	if entry.Source == "" || entry.ID == "" {
		s.renderError(w, r, fmt.Errorf("server: subtitle source and id are required"))
		return
	}
	entry.Language = lang

	subtitler := polochon.FindSubtitler(v.GetSubtitlers(), entry.Source)
	if subtitler == nil {
		s.renderError(w, r, fmt.Errorf("server: subtitler %q not found", entry.Source))
		return
	}

	sub, err := subtitler.DownloadSubtitle(v, &entry, log)
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	v.SetSubtitles([]*polochon.Subtitle{sub})

	if err := s.library.SaveSubtitles(v, log); err != nil {
		s.renderError(w, r, err)
		return
	}

	if err := s.library.UpdateSubtitleIndex(v, sub); err != nil {
		s.renderError(w, r, err)
		return
	}

	s.hub.broadcast()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listMovieSubtitles(w http.ResponseWriter, r *http.Request) {
	m := s.getMovie(w, r)
	if m == nil {
		s.renderError(w, r, index.ErrNotFound)
		return
	}
	s.listSubtitles(m, w, r)
}

func (s *Server) downloadMovieSubtitleByEntry(w http.ResponseWriter, r *http.Request) {
	m := s.getMovie(w, r)
	if m == nil {
		s.renderError(w, r, index.ErrNotFound)
		return
	}
	s.downloadSubtitleByEntry(m, w, r)
}

func (s *Server) listEpisodeSubtitles(w http.ResponseWriter, r *http.Request) {
	e := s.getEpisode(w, r)
	if e == nil {
		s.renderError(w, r, index.ErrNotFound)
		return
	}
	s.listSubtitles(e, w, r)
}

func (s *Server) downloadEpisodeSubtitleByEntry(w http.ResponseWriter, r *http.Request) {
	e := s.getEpisode(w, r)
	if e == nil {
		s.renderError(w, r, index.ErrNotFound)
		return
	}
	s.downloadSubtitleByEntry(e, w, r)
}

func getLanguage(r *http.Request) (polochon.Language, error) {
	vars := mux.Vars(r)
	return polochon.NewLanguage(vars["lang"])
}

func (s *Server) updateMovieSubtitle(w http.ResponseWriter, r *http.Request) {
	s.logEntry(r).Infof("updating movie subtitles")

	m := s.getMovie(w, r)
	if m == nil {
		s.renderError(w, r, index.ErrNotFound)
		return
	}

	s.updateSubtitle(m, w, r)
}

func (s *Server) uploadMovieSubtitle(w http.ResponseWriter, r *http.Request) {
	s.logEntry(r).Infof("uploading movie subtitles")

	m := s.getMovie(w, r)
	if m == nil {
		s.renderError(w, r, index.ErrNotFound)
		return
	}

	s.uploadSubtitle(m, w, r)
}

func (s *Server) updateEpisodeSubtitle(w http.ResponseWriter, r *http.Request) {
	s.logEntry(r).Infof("updating episode subtitles")

	e := s.getEpisode(w, r)
	if e == nil {
		s.renderError(w, r, index.ErrNotFound)
		return
	}

	s.updateSubtitle(e, w, r)
}

func (s *Server) uploadEpisodeSubtitle(w http.ResponseWriter, r *http.Request) {
	s.logEntry(r).Infof("uploading episode subtitles")

	e := s.getEpisode(w, r)
	if e == nil {
		s.renderError(w, r, index.ErrNotFound)
		return
	}

	s.uploadSubtitle(e, w, r)
}

func (s *Server) updateSubtitle(v polochon.Video, w http.ResponseWriter, r *http.Request) {
	if v == nil {
		s.renderError(w, r, index.ErrNotFound)
		return
	}
	v.SetSubtitles(nil)

	log := s.logEntry(r)

	lang, err := getLanguage(r)
	if err != nil {
		s.renderError(w, r, err)
		return
	}
	sub, err := polochon.GetSubtitle(v, lang, log)
	if err != nil {
		if err == polochon.ErrNoSubtitleFound {
			s.renderOK(w, nil)
		} else {
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

	s.hub.broadcast()
	s.renderOK(w, sub)
}

func (s *Server) uploadSubtitle(v polochon.Video, w http.ResponseWriter, r *http.Request) {
	if v == nil {
		s.renderError(w, r, index.ErrNotFound)
		return
	}
	l, err := getLanguage(r)
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	// Create Subtitle from upload
	sub := polochon.NewSubtitleFromVideo(v, l)
	sub.Data, err = io.ReadAll(r.Body)
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	if err := sub.Save(); err != nil {
		s.renderError(w, r, err)
		return
	}

	// Save in the media index
	if err := s.library.UpdateSubtitleIndex(v, sub); err != nil {
		s.renderError(w, r, err)
		return
	}

	s.hub.broadcast()
	s.renderOK(w, sub)
}

func (s *Server) serveSubtitle(v polochon.Video, w http.ResponseWriter, r *http.Request) {
	if v == nil {
		s.renderError(w, r, index.ErrNotFound)
		return
	}
	l, err := getLanguage(r)
	if err != nil {
		s.renderError(w, r, err)
		return
	}

	sub := polochon.NewSubtitleFromVideo(v, l)
	s.serveFile(w, r, &sub.File)
}

func (s *Server) serveMovieSubtitle(w http.ResponseWriter, r *http.Request) {
	m := s.getMovie(w, r)
	if m == nil {
		s.renderError(w, r, index.ErrNotFound)
		return
	}
	s.serveSubtitle(m, w, r)
}

func (s *Server) serveEpisodeSubtitle(w http.ResponseWriter, r *http.Request) {
	e := s.getEpisode(w, r)
	if e == nil {
		s.renderError(w, r, index.ErrNotFound)
		return
	}

	s.serveSubtitle(e, w, r)
}
