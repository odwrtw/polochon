package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/unrolled/render.v1"

	"github.com/odwrtw/polochon/app/auth"
	"github.com/odwrtw/polochon/app/subapp"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/library"
	index "github.com/odwrtw/polochon/lib/media_index"
	"github.com/sirupsen/logrus"
)

// AppName is the application name
const AppName = "http_server"

// Server represents a http server
type Server struct {
	*subapp.Base

	config          *configuration.Config
	library         *library.Library
	authManager     *auth.Manager
	gracefulServer  *http.Server
	shutdownCancel  context.CancelFunc
	hub    *sseHub
	log    *logrus.Entry
	render *render.Render
}

// New returns a new server
func New(config *configuration.Config, vs *library.Library, auth *auth.Manager) *Server {
	return &Server{
		Base:          subapp.NewBase(AppName),
		config:        config,
		library:       vs,
		authManager:   auth,
		hub:    newSSEHub(),
		render: render.New(),
	}
}

// Run starts the server
func (s *Server) Run(log *logrus.Entry) error {
	s.log = log.WithField("app", AppName)

	// Init the app
	s.InitStart(log)

	ctx, cancel := context.WithCancel(context.Background())
	s.shutdownCancel = cancel
	defer cancel()

	srv := s.httpServer(s.log)
	srv.BaseContext = func(_ net.Listener) context.Context { return ctx }
	s.gracefulServer = srv

	err := s.gracefulServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Stop stops the http server
func (s *Server) Stop(log *logrus.Entry) {
	if s.shutdownCancel != nil {
		s.shutdownCancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.gracefulServer.Shutdown(ctx); err != nil {
		log.WithError(err).Error("failed to shutdown http server")
	}
}

// Hub returns the SSE hub as a Notifier
func (s *Server) Hub() polochon.Notifier {
	return s.hub
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
	if file == nil || file.Size == 0 {
		s.renderError(w, r, index.ErrNotFound)
		return
	}

	filename := filepath.Base(file.Path)

	// If a name was provided in the URL, ensure it matches the actual filename
	if name := mux.Vars(r)["filename"]; name != "" && name != filename {
		s.renderError(w, r, index.ErrNotFound)
		return
	}

	s.logEntry(r).Infof("serving file %q", filename)
	// Set the header so that when downloading, the real filename will be given
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	http.ServeFile(w, r, file.Path)
}
