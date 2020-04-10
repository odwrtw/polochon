package server

import (
	"encoding/json"
	"net/http"

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

	torrent := &polochon.Torrent{}
	if err := json.NewDecoder(r.Body).Decode(torrent); err != nil {
		s.renderError(w, &Error{
			Code:    http.StatusBadRequest,
			Message: "Unable to read payload",
		})
		s.log.Warning(err.Error())
		return
	}
	if torrent.Result == nil || torrent.Result.URL == "" {
		s.renderError(w, &Error{
			Code:    http.StatusBadRequest,
			Message: "Unable to find the URL in the request",
		})
		return
	}

	if err := s.config.Downloader.Client.Download(torrent); err != nil {
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
	torrents, err := s.config.Downloader.Client.List()
	if err != nil {
		s.log.Warningf("error while listing torrents via the API: %q", err)
		s.renderError(w, &Error{
			Code:    http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	s.renderOK(w, torrents)
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
	id := vars["id"]

	// Delete the torrent
	err := s.config.Downloader.Client.Remove(&polochon.Torrent{
		Status: &polochon.TorrentStatus{
			ID: id,
		},
	})
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
