package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/media_index"
)

// Season represents the season output of the server
type Season struct {
	*polochon.ShowSeason
	Episodes []int `json:"episodes"`
}

// NewSeason returns a new season to be JSON formated
func NewSeason(season *polochon.ShowSeason, indexed index.IndexedSeason) *Season {
	return &Season{
		ShowSeason: season,
		Episodes:   indexed.EpisodeList(),
	}
}

func (s *Server) getSeasonDetails(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	seasonNum, err := strconv.Atoi(vars["season"])
	if err != nil {
		s.renderError(w, fmt.Errorf("invalid season"))
		return
	}

	season, err := s.library.GetSeason(vars["id"], seasonNum)
	if err != nil {
		s.renderError(w, err)
		return
	}

	indexedSeason, err := s.library.GetIndexedSeason(vars["id"], seasonNum)
	if err != nil {
		s.renderError(w, err)
		return
	}

	s.renderOK(w, NewSeason(season, indexedSeason))
}

func (s *Server) deleteSeason(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	seasonNum, err := strconv.Atoi(vars["season"])
	if err != nil {
		s.renderError(w, fmt.Errorf("invalid season"))
		return
	}

	if err := s.library.DeleteSeason(vars["id"], seasonNum, s.log); err != nil {
		s.renderError(w, err)
		return
	}

	s.renderOK(w, nil)
}
