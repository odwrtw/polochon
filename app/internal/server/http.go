package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"gopkg.in/unrolled/render.v1"

	"github.com/Sirupsen/logrus"
	"github.com/braintree/manners"
	"github.com/gorilla/mux"
	"github.com/odwrtw/polochon/app/internal/configuration"
	"github.com/odwrtw/polochon/app/internal/subapp"
	"github.com/odwrtw/polochon/app/internal/token"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/library"
	"github.com/odwrtw/polochon/lib/media_index"
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

func (s *Server) movieIds(w http.ResponseWriter, req *http.Request) {
	s.log.Debug("listing movies by ids")

	movieIds, err := s.library.MovieIds()
	if err != nil {
		s.renderError(w, err)
		return
	}
	s.renderOK(w, movieIds)
}

func (s *Server) getMovieDetails(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]

	s.log.Debugf("looking for a movie with ID %q", id)

	// Find the file
	v, err := s.library.SearchMovieByImdbID(id)
	if err != nil {
		s.log.Error(err)
		var status int
		if err == index.ErrNotFound {
			status = http.StatusNotFound
		} else {
			status = http.StatusInternalServerError
		}
		s.renderError(w, &Error{
			Code:    status,
			Message: "URL not found",
		})
		return
	}

	s.renderOK(w, v)
}

func (s *Server) showIds(w http.ResponseWriter, req *http.Request) {
	s.log.Debug("listing shows")

	ids, err := s.library.ShowIds()
	if err != nil {
		s.renderError(w, err)
		return
	}

	ret := map[string]map[string][]string{}
	for id, show := range ids {
		ret[id] = map[string][]string{}
		for seasonNum, season := range show.Seasons {
			s := fmt.Sprintf("%02d", seasonNum)
			for episode := range season.Episodes {
				e := fmt.Sprintf("%02d", episode)

				if _, ok := ret[id][s]; !ok {
					ret[id][s] = []string{}
				}

				ret[id][s] = append(ret[id][s], e)
			}
		}
	}

	s.renderOK(w, ret)
}

func (s *Server) getShowDetails(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	v, err := s.library.NewShowFromID(vars["id"])
	if err != nil {
		s.log.Error(err)
		var status int
		if err == index.ErrNotFound {
			status = http.StatusNotFound
		} else {
			status = http.StatusInternalServerError
		}
		s.renderError(w, &Error{
			Code:    status,
			Message: "URL not found",
		})
		return
	}

	s.renderOK(w, v)
}

func (s *Server) getShowEpisodeIDDetails(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	var season, episode int
	for ptr, str := range map[*int]string{
		&season:  vars["season"],
		&episode: vars["episode"],
	} {
		v, err := strconv.Atoi(str)
		if err != nil {
			s.renderError(w, fmt.Errorf("invalid season or episode"))
			return
		}
		*ptr = v
	}

	v, err := s.library.SearchShowEpisodeByImdbID(vars["id"], season, episode)
	if err != nil {
		s.log.Error(err)
		var status int
		if err == index.ErrNotFound {
			status = http.StatusNotFound
		} else {
			status = http.StatusInternalServerError
		}
		s.renderError(w, &Error{
			Code:    status,
			Message: "URL not found",
		})
		return
	}

	s.renderOK(w, v)
}

func (s *Server) wishlist(w http.ResponseWriter, req *http.Request) {
	wl := polochon.NewWishlist(s.config.Wishlist, s.log)

	if err := wl.Fetch(); err != nil {
		s.renderError(w, err)
		return
	}

	s.renderOK(w, wl)
}

func serveFile(w http.ResponseWriter, r *http.Request, file *polochon.File) {
	// Set the header so that when downloading, the real filename will be given
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(file.Path)))
	http.ServeFile(w, r, file.Path)
}

func (s *Server) serveMovie(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	v, err := s.library.SearchMovieByImdbID(vars["id"])
	s.serveVideo(w, req, v, err)
}

func (s *Server) serveShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	if vars["id"] == "" && vars["season"] == "" && vars["episode"] == "" {
		s.renderError(w, &Error{
			Code:    http.StatusNotFound,
			Message: "URL not found",
		})
	}

	season, err := strconv.Atoi(vars["season"])
	if err != nil {
		s.renderError(w, err)
		return
	}

	episode, err := strconv.Atoi(vars["episode"])
	if err != nil {
		s.renderError(w, err)
		return
	}

	v, err := s.library.SearchShowEpisodeByImdbID(vars["id"], season, episode)
	s.serveVideo(w, r, v, err)
}

func (s *Server) serveVideo(w http.ResponseWriter, r *http.Request, v polochon.Video, err error) {
	if err != nil {
		s.log.Error(err)
		var status int
		if err == index.ErrNotFound {
			status = http.StatusNotFound
		} else {
			status = http.StatusInternalServerError
		}
		s.renderError(w, &Error{
			Code:    status,
			Message: err.Error(),
		})
		return
	}

	serveFile(w, r, v.GetFile())
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
		s.log.Warning("Error while adding a torrent via the API: %q", err)
		s.renderError(w, &Error{
			Code:    http.StatusBadRequest,
			Message: "Unkown error",
		})
		return
	}
}

func (s *Server) tokenGetAllowed(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	allowed := s.tokenManager.GetAllowed(token)
	s.renderOK(w, allowed)
}
