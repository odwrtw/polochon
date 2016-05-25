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
)

// AppName is the application name
const AppName = "http_server"

// Server represents a http server
type Server struct {
	*subapp.Base

	config         *configuration.Config
	videoStore     *polochon.VideoStore
	tokenManager   *token.Manager
	gracefulServer *manners.GracefulServer
	log            *logrus.Entry
	render         *render.Render
}

// New returns a new server
func New(config *configuration.Config, vs *polochon.VideoStore, tm *token.Manager) *Server {
	return &Server{
		Base:         subapp.NewBase(AppName),
		config:       config,
		videoStore:   vs,
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

func (s *Server) movieSlugs(w http.ResponseWriter, req *http.Request) {
	s.log.Debug("listing movies by slugs")

	movieSlugs, err := s.videoStore.MovieSlugs()
	if err != nil {
		s.renderError(w, err)
		return
	}
	s.renderOK(w, movieSlugs)
}

func (s *Server) movieIds(w http.ResponseWriter, req *http.Request) {
	s.log.Debug("listing movies by ids")

	movieIds, err := s.videoStore.MovieIds()
	if err != nil {
		s.renderError(w, err)
		return
	}
	s.renderOK(w, movieIds)
}

func (s *Server) getMovieDetails(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	idType := vars["idType"]
	id := vars["id"]

	s.log.Debugf("looking for a movie by %q with ID %q", idType, id)

	var searchFunc func(id string) (polochon.Video, error)
	switch idType {
	case "ids":
		searchFunc = s.videoStore.SearchMovieByImdbID
	case "slugs":
		searchFunc = s.videoStore.SearchMovieBySlug
	default:
		s.renderError(w, fmt.Errorf("invalid id type: %q", idType))
	}

	// Find the file by Slug
	v, err := searchFunc(id)
	if err != nil {
		s.log.Error(err)
		var status int
		if err == polochon.ErrSlugNotFound {
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

	ids, err := s.videoStore.ShowIds()
	if err != nil {
		s.renderError(w, err)
		return
	}

	// JSON only allows strings as keys, the ids must me converted from int to
	// string
	ret := map[string]map[string][]string{}
	for imdbID, seasons := range ids {
		ret[imdbID] = map[string][]string{}
		for season, episodes := range seasons {
			s := fmt.Sprintf("%02d", season)
			for episode := range episodes {
				e := fmt.Sprintf("%02d", episode)

				if _, ok := ret[imdbID][s]; !ok {
					ret[imdbID][s] = []string{}
				}

				ret[imdbID][s] = append(ret[imdbID][s], e)
			}
		}
	}

	s.renderOK(w, ret)
}

func (s *Server) showSlugs(w http.ResponseWriter, req *http.Request) {
	s.log.Debug("listing shows by slugs")

	slugs, err := s.videoStore.ShowSlugs()
	if err != nil {
		s.renderError(w, err)
		return
	}

	s.renderOK(w, slugs)
}

func (s *Server) getShowEpisodeSlugDetails(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	slug := vars["slug"]

	v, err := s.videoStore.SearchShowEpisodeBySlug(slug)
	if err != nil {
		s.log.Error(err)
		var status int
		if err == polochon.ErrSlugNotFound {
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
	id := vars["id"]
	seasonStr := vars["season"]
	episodeStr := vars["episode"]

	var season, episode int
	for ptr, str := range map[*int]string{
		&season:  seasonStr,
		&episode: episodeStr,
	} {
		v, err := strconv.Atoi(str)
		if err != nil {
			s.renderError(w, fmt.Errorf("invalid season or episode"))
			return
		}
		*ptr = v
	}

	v, err := s.videoStore.SearchShowEpisodeByImdbID(id, season, episode)
	if err != nil {
		s.log.Error(err)
		var status int
		if err == polochon.ErrImdbIDNotFound {
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
	switch {
	case vars["slug"] != "":
		v, err := s.videoStore.SearchMovieBySlug(vars["slug"])
		s.serveVideo(w, req, v, err)
	case vars["id"] != "":
		v, err := s.videoStore.SearchMovieByImdbID(vars["id"])
		s.serveVideo(w, req, v, err)
	default:
		s.renderError(w, &Error{
			Code:    http.StatusNotFound,
			Message: "URL not found",
		})
	}
}

func (s *Server) serveShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	switch {
	case vars["slug"] != "":
		v, err := s.videoStore.SearchShowEpisodeBySlug(vars["slug"])
		s.serveVideo(w, r, v, err)
	case vars["id"] != "" && vars["season"] != "" && vars["episode"] != "":
		sStr := vars["season"]
		eStr := vars["episode"]
		season, err := strconv.Atoi(sStr)
		episode, err := strconv.Atoi(eStr)
		if err != nil {
			s.renderError(w, err)
			return
		}
		v, err := s.videoStore.SearchShowEpisodeByImdbID(vars["id"], season, episode)
		s.serveVideo(w, r, v, err)
	default:
		s.renderError(w, &Error{
			Code:    http.StatusNotFound,
			Message: "URL not found",
		})
	}
}

func (s *Server) serveVideo(w http.ResponseWriter, r *http.Request, v polochon.Video, err error) {
	if err != nil {
		s.log.Error(err)
		var status int
		if err == polochon.ErrSlugNotFound {
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

func (s *Server) deleteFile(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	videoType := vars["videoType"]
	slug := vars["slug"]

	s.log.Debugf("looking for the %s: %s", videoType, slug)

	var searchFunc func(slug string) (polochon.Video, error)
	switch videoType {
	case "movies":
		searchFunc = s.videoStore.SearchMovieBySlug
	case "shows":
		searchFunc = s.videoStore.SearchShowEpisodeBySlug
	default:
		s.renderError(w, fmt.Errorf("invalid video type: %q", videoType))
		return
	}

	// Find the file by Slug
	v, err := searchFunc(slug)
	if err != nil {
		s.log.Error(err)
		var status int
		if err == polochon.ErrSlugNotFound {
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

	videoFile := v.GetFile()
	s.log.Debugf("got the file to delete: %s", filepath.Base(videoFile.Path))

	err = s.videoStore.Delete(v, s.log)
	if err != nil {
		s.log.Errorf("failed to delete video : %q", err)
		s.renderError(w, err)
	}

	s.renderOK(w, nil)
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
