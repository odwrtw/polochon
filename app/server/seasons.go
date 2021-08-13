package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	polochon "github.com/odwrtw/polochon/lib"
	index "github.com/odwrtw/polochon/lib/media_index"
)

// Season represents the season output of the server
type Season struct {
	*polochon.ShowSeason
	Episodes map[int]*index.Episode `json:"episodes"`
}

// NewSeason returns a new season to be JSON formated
func NewSeason(season *polochon.ShowSeason, indexed *index.Season) *Season {
	return &Season{
		ShowSeason: season,
		Episodes:   indexed.Episodes,
	}
}

func (s *Server) getSeasonDetails(w http.ResponseWriter, req *http.Request) {
	s.logEntry(req).Infof("getting season details")
	vars := mux.Vars(req)

	seasonNum, err := strconv.Atoi(vars["season"])
	if err != nil {
		s.renderError(w, req, fmt.Errorf("invalid season"))
		return
	}

	season, err := s.library.GetSeason(vars["id"], seasonNum)
	if err != nil {
		s.renderError(w, req, err)
		return
	}

	indexedSeason, err := s.library.GetIndexedSeason(vars["id"], seasonNum)
	if err != nil {
		s.renderError(w, req, err)
		return
	}

	s.renderOK(w, NewSeason(season, indexedSeason))
}

func (s *Server) deleteSeason(w http.ResponseWriter, req *http.Request) {
	log := s.logEntry(req)
	log.Infof("deleting season details")
	vars := mux.Vars(req)

	seasonNum, err := strconv.Atoi(vars["season"])
	if err != nil {
		s.renderError(w, req, fmt.Errorf("invalid season"))
		return
	}

	if err := s.library.DeleteSeason(vars["id"], seasonNum, log); err != nil {
		s.renderError(w, req, err)
		return
	}

	s.renderOK(w, nil)
}
