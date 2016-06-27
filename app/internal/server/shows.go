package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/media_index"
)

// Show represents the show output of the server
type Show struct {
	*polochon.Show
	Seasons []int `json:"seasons"`
}

// NewShow returns a new show to be JSON formated
func NewShow(show *polochon.Show, indexed index.IndexedShow) *Show {
	return &Show{
		Show:    show,
		Seasons: indexed.SeasonList(),
	}
}

func (s *Server) showIds(w http.ResponseWriter, req *http.Request) {
	s.log.Debug("listing shows")

	ret := map[string]map[string][]string{}
	for id, show := range s.library.ShowIDs() {
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

	s.renderOK(w, NewShow(show, indexedShow))
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

func (s *Server) serveShow(w http.ResponseWriter, r *http.Request) {
	e := s.getEpisode(w, r)
	if e == nil {
		return
	}

	s.serveFile(w, r, e.GetFile())
}
