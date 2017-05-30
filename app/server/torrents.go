package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	polochon "github.com/odwrtw/polochon/lib"
)

func (s *Server) addTorrent(w http.ResponseWriter, r *http.Request) {
	if !s.config.Downloader.Enabled {
		s.log.Warning("downloader not available")
		s.renderError(w, &Error{
			Code:    http.StatusServiceUnavailable,
			Message: "downloader not enabled in your polochon",
		})
		return
	}

	req := new(struct{ URL string })
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		s.renderError(w, &Error{
			Code:    http.StatusBadRequest,
			Message: "Unable to read payload",
		})
		s.log.Warning(err.Error())
		return
	}
	if req.URL == "" {
		s.renderError(w, &Error{
			Code:    http.StatusBadRequest,
			Message: "Unable to find the URL in the request",
		})
		return
	}

	if err := s.config.Downloader.Client.Download(req.URL, s.log.WithField("function", "downloader")); err != nil {
		if err == polochon.ErrDuplicateTorrent {
			s.renderError(w, &Error{
				Code:    http.StatusConflict,
				Message: "Torrent already added",
			})
			return
		}
		s.log.Warningf("Error while adding a torrent via the API: %q", err)
		s.renderError(w, &Error{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}
	s.renderOK(w, nil)
}

func (s *Server) getTorrents(w http.ResponseWriter, r *http.Request) {
	// Check that the downloader is enabled
	if !s.config.Downloader.Enabled {
		s.log.Warning("downloader not available")
		s.renderError(w, &Error{
			Code:    http.StatusServiceUnavailable,
			Message: "downloader not enabled in your polochon",
		})
		return
	}

	// Get the list of the ongoing torrents
	list, err := s.config.Downloader.Client.List()
	if err != nil {
		s.log.Warningf("error while listing torrents via the API: %q", err)
		s.renderError(w, &Error{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	// Fill an array of DownloadableInfos
	result := []*polochon.DownloadableInfos{}
	for _, t := range list {
		result = append(result, t.Infos())
	}

	// render the response
	s.renderOK(w, result)
}

func (s *Server) removeTorrent(w http.ResponseWriter, r *http.Request) {
	// Check that the downloader is enabled
	if !s.config.Downloader.Enabled {
		s.log.Warning("downloader not available")
		s.renderError(w, &Error{
			Code:    http.StatusServiceUnavailable,
			Message: "downloader not enabled in your polochon",
		})
		return
	}

	// Get the torrent ID from the URL
	vars := mux.Vars(r)
	torrentID, err := strconv.Atoi(vars["torrentID"])
	if err != nil {
		s.renderError(w, &Error{
			Code:    http.StatusBadRequest,
			Message: "Invalid torrentID",
		})
		return
	}

	// List all the ongoing torrents
	list, err := s.config.Downloader.Client.List()
	if err != nil {
		s.renderError(w, &Error{
			Code:    http.StatusInternalServerError,
			Message: "Unable to list torrents",
		})
		s.log.Errorf("error while getting torrent list: %q", err)
		return
	}

	var torrent *polochon.Downloadable
	// Iterate over the list, looking for the torrent to delete
	for _, t := range list {
		// Fetch the infos of the given torrent
		torrentInfos := t.Infos()
		if torrentInfos == nil {
			s.log.Warn("got nil Infos while getting torrent infos")
			continue
		}
		id, ok := torrentInfos.AdditionalInfos["id"].(int)
		if !ok {
			s.log.Warn("got invalid ID in torrent Infos")
			continue
		}
		if id == torrentID {
			torrent = &t
			break
		}
	}

	// If we didn't find the torrent, return an error
	if torrent == nil {
		s.renderError(w, &Error{
			Code:    http.StatusNotFound,
			Message: "No such torrent",
		})
		return
	}

	// Delete the torrent
	err = s.config.Downloader.Client.Remove(*torrent)
	if err != nil {
		s.log.Warningf("error while removing torrent: %q", err)
		s.renderError(w, &Error{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	// render the response
	s.renderOK(w, nil)
}
