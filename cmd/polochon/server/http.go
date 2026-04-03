package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/unrolled/render.v1"

	"github.com/odwrtw/polochon/cmd/polochon/auth"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/library"
	index "github.com/odwrtw/polochon/lib/media_index"
)

// AppName is the application name
const AppName = "http_server"

// Server represents a http server
type Server struct {
	config      *configuration.Config
	library     *library.Library
	authManager *auth.Manager
	hub         *sseHub
	log         *slog.Logger
	render      *render.Render
}

// New returns a new server
func New(config *configuration.Config, vs *library.Library, auth *auth.Manager, log *slog.Logger) *Server {
	return &Server{
		config:      config,
		library:     vs,
		authManager: auth,
		hub:         newSSEHub(),
		log:         log.With("app", AppName),
		render:      render.New(),
	}
}

// Run starts the server
func (s *Server) Run(ctx context.Context) error {
	srv := s.httpServer()
	srv.BaseContext = func(_ net.Listener) context.Context { return ctx }

	serveErr := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveErr <- err
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			s.log.Error("failed to shutdown http server", "error", err)
		}
		return <-serveErr
	}
}

// Hub returns the SSE hub as a Notifier
func (s *Server) Hub() polochon.Notifier {
	return s.hub
}

func (s *Server) wishlist(w http.ResponseWriter, r *http.Request) {
	log := s.logEntry(r)
	log.Info("getting wishlist")

	wl := polochon.NewWishlist(s.config.Wishlist, log)

	if err := wl.Fetch(r.Context()); err != nil {
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

	s.logEntry(r).Info("serving file", "filename", filename)
	// Set the header so that when downloading, the real filename will be given
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	http.ServeFile(w, r, file.Path)
}
