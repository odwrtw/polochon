package server

import (
	"encoding/json"
	"fmt"
	"net/http"

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
		s.renderError(w, fmt.Errorf("Unkown error"))
		s.log.Warning(err.Error())
		return
	}
	if req.URL == "" {
		s.renderError(w, &Error{
			Code:    http.StatusBadRequest,
			Message: "Unkown error",
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
