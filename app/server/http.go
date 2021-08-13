package server

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	"gopkg.in/unrolled/render.v1"

	"github.com/odwrtw/polochon/app/auth"
	"github.com/odwrtw/polochon/app/subapp"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/library"
	"github.com/sirupsen/logrus"
)

// AppName is the application name
const AppName = "http_server"

// Server represents a http server
type Server struct {
	*subapp.Base

	config         *configuration.Config
	library        *library.Library
	authManager    *auth.Manager
	gracefulServer *http.Server
	log            *logrus.Entry
	render         *render.Render
}

// New returns a new server
func New(config *configuration.Config, vs *library.Library, auth *auth.Manager) *Server {
	return &Server{
		Base:        subapp.NewBase(AppName),
		config:      config,
		library:     vs,
		authManager: auth,
		render:      render.New(),
	}
}

// Run starts the server
func (s *Server) Run(log *logrus.Entry) error {
	s.log = log.WithField("app", AppName)

	// Init the app
	s.InitStart(log)

	s.gracefulServer = s.httpServer(s.log)
	err := s.gracefulServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Stop stops the http server
func (s *Server) Stop(log *logrus.Entry) {
	s.gracefulServer.Shutdown(context.Background())
}

func (s *Server) wishlist(w http.ResponseWriter, r *http.Request) {
	log := s.logEntry(r)
	log.Infof("getting wishlist")

	wl := polochon.NewWishlist(s.config.Wishlist, log)

	if err := wl.Fetch(); err != nil {
		s.renderError(w, r, err)
		return
	}

	s.renderOK(w, wl)
}

func (s *Server) serveFile(w http.ResponseWriter, r *http.Request, file *polochon.File) {
	filename := filepath.Base(file.Path)
	s.logEntry(r).Infof("serving file %q", filename)
	// Set the header so that when downloading, the real filename will be given
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	http.ServeFile(w, r, file.Path)
}

func (s *Server) tokenGetAllowed(w http.ResponseWriter, r *http.Request) {
	s.logEntry(r).Infof("getting tokens")
	token := r.URL.Query().Get("token")
	allowed := s.authManager.GetAllowed(token)
	s.renderOK(w, allowed)
}
