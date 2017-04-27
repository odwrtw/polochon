package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/odwrtw/polochon/lib"
	index "github.com/odwrtw/polochon/lib/media_index"
)

// Format seasons to get a pretty marshal
func formatSeasons(s index.IndexedShow) map[string][]string {
	ret := map[string][]string{}
	for seasonNum, season := range s.Seasons {
		s := fmt.Sprintf("%02d", seasonNum)
		for episode := range season.Episodes {
			e := fmt.Sprintf("%02d", episode)

			if _, ok := ret[s]; !ok {
				ret[s] = []string{}
			}

			ret[s] = append(ret[s], e)
		}
	}
	return ret
}

func (s *Server) showIds(w http.ResponseWriter, req *http.Request) {
	s.log.Debug("listing shows")

	ret := map[string]map[string][]string{}
	for id, show := range s.library.ShowIDs() {
		ret[id] = formatSeasons(show)
	}

	s.renderOK(w, ret)
}

// TODO: handle this in a middleware
func (s *Server) getEpisode(w http.ResponseWriter, req *http.Request) *polochon.ShowEpisode {
	vars := mux.Vars(req)

	var season, episode int
	for ptr, str := range map[*int]string{
		&season:  vars["season"],
		&episode: vars["episode"],
	} {
		v, err := strconv.Atoi(str)
		if err != nil {
			s.renderError(w, fmt.Errorf("invalid season or episode"))
			return nil
		}
		*ptr = v
	}

	e, err := s.library.GetEpisode(vars["id"], season, episode)
	if err != nil {
		s.renderError(w, err)
		return nil
	}

	return e
}

func (s *Server) getShowDetails(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	show, err := s.library.GetShow(vars["id"])
	if err != nil {
		s.renderError(w, err)
		return
	}

	indexedShow, err := s.library.GetIndexedShow(vars["id"])
	if err != nil {
		s.renderError(w, err)
		return
	}

	out := struct {
		*polochon.Show
		Seasons map[string][]string `json:"seasons"`
	}{
		show,
		formatSeasons(indexedShow),
	}

	s.renderOK(w, out)
}

func (s *Server) deleteShow(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	if err := s.library.DeleteShow(vars["id"], s.log); err != nil {
		s.renderError(w, err)
		return
	}

	s.renderOK(w, nil)
}

func (s *Server) getShowEpisodeIDDetails(w http.ResponseWriter, req *http.Request) {
	e := s.getEpisode(w, req)
	if e == nil {
		return
	}

	s.renderOK(w, e)
}

func (s *Server) deleteEpisode(w http.ResponseWriter, req *http.Request) {
	e := s.getEpisode(w, req)
	if e == nil {
		return
	}

	if err := s.library.Delete(e, s.log); err != nil {
		s.renderError(w, err)
		return
	}

	s.renderOK(w, nil)
}

func (s *Server) updateEpisodeSubtitles(w http.ResponseWriter, req *http.Request) {
	e := s.getEpisode(w, req)
	if e == nil {
		return
	}

	if err := s.library.AddSubtitles(e, s.config.SubtitleLanguages, s.log); err != nil {
		s.renderError(w, err)
		return
	}

	s.renderOK(w, nil)
}

func (s *Server) serveShow(w http.ResponseWriter, r *http.Request) {
	e := s.getEpisode(w, r)
	if e == nil {
		return
	}

	s.serveFile(w, r, e.GetFile())
}
