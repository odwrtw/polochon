package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/media_index"
)

func (s *Server) showIds(w http.ResponseWriter, req *http.Request) {
	s.log.Debug("listing shows")

	ids, err := s.library.ShowIDs()
	if err != nil {
		s.renderError(w, err)
		return
	}

	ret := map[string]map[string][]string{}
	for id, show := range ids {
		ret[id] = map[string][]string{}
		for seasonNum, season := range show.Seasons {
			s := fmt.Sprintf("%02d", seasonNum)
			for episode := range season.Episodes {
				e := fmt.Sprintf("%02d", episode)

				if _, ok := ret[id][s]; !ok {
					ret[id][s] = []string{}
				}

				ret[id][s] = append(ret[id][s], e)
			}
		}
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

	return e
}

func (s *Server) getShowDetails(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	v, err := s.library.NewShowFromID(vars["id"])
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
		return
	}

	s.renderOK(w, v)
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
		s.renderError(w, &Error{
			Code:    http.StatusInternalServerError,
			Message: "URL not found",
		})
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
