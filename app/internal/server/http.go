package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

	"gopkg.in/unrolled/render.v1"

	"github.com/Sirupsen/logrus"
	"github.com/braintree/manners"
	"github.com/odwrtw/polochon/app/internal/configuration"
	"github.com/odwrtw/polochon/app/internal/subapp"
	"github.com/odwrtw/polochon/app/internal/token"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/library"
)

// AppName is the application name
const AppName = "http_server"

// Server represents a http server
type Server struct {
	*subapp.Base

	config         *configuration.Config
	library        *library.Library
	tokenManager   *token.Manager
	gracefulServer *manners.GracefulServer
	log            *logrus.Entry
	render         *render.Render
}

// New returns a new server
func New(config *configuration.Config, vs *library.Library, tm *token.Manager) *Server {
	return &Server{
		Base:         subapp.NewBase(AppName),
		config:       config,
		library:      vs,
		tokenManager: tm,
		render:       render.New(),
	}
}

// Run starts the server
func (s *Server) Run(log *logrus.Entry) error {
	s.log = log.WithField("app", AppName)

	// Init the app
	s.InitStart(log)

	s.gracefulServer = manners.NewWithServer(s.httpServer(s.log))
	return s.gracefulServer.ListenAndServe()
}

// Stop stops the http server
func (s *Server) Stop(log *logrus.Entry) {
	s.gracefulServer.Close()
}

// BlockingStop stops the http server and waits for it to be done
func (s *Server) BlockingStop(log *logrus.Entry) {
	s.gracefulServer.BlockingClose()
}

func (s *Server) wishlist(w http.ResponseWriter, req *http.Request) {
	wl := polochon.NewWishlist(s.config.Wishlist, s.log)

	if err := wl.Fetch(); err != nil {
		s.renderError(w, err)
		return
	}

	s.renderOK(w, wl)
}

func (s *Server) serveFile(w http.ResponseWriter, r *http.Request, file *polochon.File) {
	// Set the header so that when downloading, the real filename will be given
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(file.Path)))
	http.ServeFile(w, r, file.Path)
}

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

func (s *Server) tokenGetAllowed(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	allowed := s.tokenManager.GetAllowed(token)
	s.renderOK(w, allowed)
}
