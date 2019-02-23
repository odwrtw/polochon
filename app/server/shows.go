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
func formatSeasons(show *index.Show) map[string]map[string]*index.Episode {
	ret := map[string]map[string]*index.Episode{}
	for seasonNum, season := range show.Seasons {
		s := fmt.Sprintf("%02d", seasonNum)
		for episodeNb, episode := range season.Episodes {
			e := fmt.Sprintf("%02d", episodeNb)

			if _, ok := ret[s]; !ok {
				ret[s] = map[string]*index.Episode{}
			}

			ret[s][e] = episode
		}
	}
	return ret
}

func (s *Server) showIds(w http.ResponseWriter, req *http.Request) {
	s.log.Debug("listing shows")

	type formatedShow struct {
		Title   string                               `json:"title"`
		Seasons map[string]map[string]*index.Episode `json:"seasons"`
	}
	ret := map[string]formatedShow{}
	for id, show := range s.library.ShowIDs() {
		ret[id] = formatedShow{
			show.Title,
			formatSeasons(show),
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

	out := struct {
		*polochon.Show
		Seasons map[string]map[string]*index.Episode `json:"seasons"`
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

	idxEpisode, err := s.library.GetIndexedEpisode(e.ShowImdbID, e.Season, e.Episode)
	if err != nil {
		s.renderError(w, err)
		return
	}

	episode := struct {
		*polochon.ShowEpisode
		Subtitles []polochon.Language `json:"subtitles"`
	}{
		ShowEpisode: e,
		Subtitles:   idxEpisode.Subtitles,
	}

	s.renderOK(w, episode)
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

func (s *Server) serveEpisode(w http.ResponseWriter, req *http.Request) {
	e := s.getEpisode(w, req)
	if e == nil {
		return
	}

	s.serveFile(w, req, e.GetFile())
}
