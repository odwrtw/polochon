package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	polochon "github.com/odwrtw/polochon/lib"
	index "github.com/odwrtw/polochon/lib/media_index"
)

func (s *Server) getShowFiles(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	imdbID := vars["id"]
	name := vars["name"]

	show, err := s.library.GetIndexedShow(imdbID)
	if err != nil {
		s.renderError(w, req, err)
		return
	}

	var path string
	for _, f := range []*polochon.File{show.FanartFile, show.BannerFile, show.PosterFile, show.NFOFile} {
		if f != nil && f.Name == name {
			path = f.Path
			break
		}
	}

	if path == "" {
		s.renderError(w, req, index.ErrNotFound)
		return
	}

	s.serveFile(w, req, polochon.NewFile(path))
}

func (s *Server) getSeasonDetails(w http.ResponseWriter, req *http.Request) {
	s.logEntry(req).Info("getting season details")
	vars := mux.Vars(req)

	seasonNum, err := strconv.Atoi(vars["season"])
	if err != nil {
		s.renderError(w, req, fmt.Errorf("invalid season"))
		return
	}

	season, err := s.library.GetIndexedSeason(vars["id"], seasonNum)
	if err != nil {
		s.renderError(w, req, err)
		return
	}

	s.renderOK(w, season)
}

func (s *Server) deleteSeason(w http.ResponseWriter, req *http.Request) {
	log := s.logEntry(req)
	log.Info("deleting season details")
	vars := mux.Vars(req)

	seasonNum, err := strconv.Atoi(vars["season"])
	if err != nil {
		s.renderError(w, req, fmt.Errorf("invalid season"))
		return
	}

	if err := s.library.DeleteSeason(vars["id"], seasonNum); err != nil {
		s.renderError(w, req, err)
		return
	}

	s.hub.broadcast()
	s.renderOK(w, nil)
}
