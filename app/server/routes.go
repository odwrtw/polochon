package server

import (
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/gorilla/mux"
	"github.com/odwrtw/polochon/app/auth"
	"github.com/phyber/negroni-gzip/gzip"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/urfave/negroni"
)

// httpServer returns an http server
func (s *Server) httpServer(log *logrus.Entry) *http.Server {
	addr := fmt.Sprintf("%s:%d", s.config.HTTPServer.Host, s.config.HTTPServer.Port)
	log.Debugf("http server will listen on: %s", addr)

	mux := mux.NewRouter()
	for _, route := range []struct {
		// name of the route
		name string
		// path of the route
		path string
		// allowed methods for this route
		methods string
		// handler is the http handler to run if the route matches
		handler func(http.ResponseWriter, *http.Request)
		// excluded tells if the route should be added to the router,
		// it's in the negative form so that the default behaviour is to add
		// the route to the router
		excluded bool
	}{
		{
			name:    "GetMovies",
			path:    "/movies",
			methods: "GET",
			handler: s.movieIndex,
		},
		{
			name:    "GetMovie",
			path:    "/movies/{id}",
			methods: "GET",
			handler: s.getMovieDetails,
		},
		{
			name:    "DeleteMovie",
			path:    "/movies/{id}",
			methods: "DELETE",
			handler: s.deleteMovie,
		},
		{
			name:     "DownloadMovie",
			path:     "/movies/{id}/download",
			methods:  "GET",
			handler:  s.serveMovie,
			excluded: !s.config.HTTPServer.ServeFiles,
		},
		{
			name:     "DownloadMovieFiles",
			path:     "/movies/{id}/files/{name}",
			methods:  "GET",
			handler:  s.serveMovieFile,
			excluded: !s.config.HTTPServer.ServeFiles,
		},
		{
			name:     "DownloadMovieSubtitle",
			path:     "/movies/{id}/subtitles/{lang}/download",
			methods:  "GET",
			handler:  s.serveMovieSubtitle,
			excluded: !s.config.HTTPServer.ServeFiles,
		},
		{
			name:    "UpdateMovieSubtitle",
			path:    "/movies/{id}/subtitles/{lang}",
			methods: "POST",
			handler: s.updateMovieSubtitle,
		},
		{
			name:    "UploadMovieSubtitle",
			path:    "/movies/{id}/subtitles/{lang}",
			methods: "PUT",
			handler: s.uploadMovieSubtitle,
		},
		{
			name:    "GetShows",
			path:    "/shows",
			methods: "GET",
			handler: s.showIds,
		},
		{
			name:    "GetShow",
			path:    "/shows/{id}",
			methods: "GET",
			handler: s.getShowDetails,
		},
		{
			name:    "DeleteShow",
			path:    "/shows/{id}",
			methods: "DELETE",
			handler: s.deleteShow,
		},
		{
			name:    "DownloadShowFiles",
			path:    "/shows/{id}/files/{name}",
			methods: "GET",
			handler: s.getShowFiles,
		},
		{
			name:    "GetSeason",
			path:    "/shows/{id}/seasons/{season:[0-9]+}",
			methods: "GET",
			handler: s.getSeasonDetails,
		},
		{
			name:    "DeleteSeason",
			path:    "/shows/{id}/seasons/{season:[0-9]+}",
			methods: "DELETE",
			handler: s.deleteSeason,
		},
		{
			name:    "GetEpisode",
			path:    "/shows/{id}/seasons/{season:[0-9]+}/episodes/{episode:[0-9]+}",
			methods: "GET",
			handler: s.getShowEpisodeIDDetails,
		},
		{
			name:    "DeleteEpisode",
			path:    "/shows/{id}/seasons/{season:[0-9]+}/episodes/{episode:[0-9]+}",
			methods: "DELETE",
			handler: s.deleteEpisode,
		},
		{
			name:    "UpdateEpisodeSubtitle",
			path:    "/shows/{id}/seasons/{season:[0-9]+}/episodes/{episode:[0-9]+}/subtitles/{lang}",
			methods: "POST",
			handler: s.updateEpisodeSubtitle,
		},
		{
			name:    "UploadEpisodeSubtitle",
			path:    "/shows/{id}/seasons/{season:[0-9]+}/episodes/{episode:[0-9]+}/subtitles/{lang}",
			methods: "PUT",
			handler: s.uploadEpisodeSubtitle,
		},
		{
			name:     "DownloadEpisode",
			path:     "/shows/{id}/seasons/{season:[0-9]+}/episodes/{episode:[0-9]+}/download",
			methods:  "GET",
			handler:  s.serveEpisode,
			excluded: !s.config.HTTPServer.ServeFiles,
		},
		{
			name:    "DownloadEpisodeFiles",
			path:    "/shows/{id}/seasons/{season:[0-9]+}/episodes/{episode:[0-9]+}/files/{name}",
			methods: "GET",
			handler: s.getShowEpisodeFiles,
		},
		{
			name:     "DownloadEpisodeSubtitle",
			path:     "/shows/{id}/seasons/{season:[0-9]+}/episodes/{episode:[0-9]+}/subtitles/{lang}/download",
			methods:  "GET",
			handler:  s.serveEpisodeSubtitle,
			excluded: !s.config.HTTPServer.ServeFiles,
		},
		{
			name:    "Wishlist",
			path:    "/wishlist",
			methods: "GET",
			handler: s.wishlist,
		},
		{
			name:    "AddTorrent",
			path:    "/torrents",
			methods: "POST",
			handler: s.addTorrent,
		},
		{
			name:    "ListTorrents",
			path:    "/torrents",
			methods: "GET",
			handler: s.getTorrents,
		},
		{
			name:    "RemoveTorrent",
			path:    "/torrents/{id}",
			methods: "DELETE",
			handler: s.removeTorrent,
		},
		{
			name:    "LibraryRefresh",
			path:    "/library/refresh",
			methods: "POST",
			handler: s.libraryRefresh,
		},
		{
			name:    "GetModulesStatus",
			path:    "/modules/status",
			methods: "GET",
			handler: s.getModulesStatus,
		},
		{
			name:    "PprofIndex",
			path:    "/debug/pprof/",
			methods: "GET",
			handler: pprof.Index,
		},
		{
			name:    "PprofBlock",
			path:    "/debug/pprof/block",
			methods: "GET",
			handler: pprof.Index,
		},
		{
			name:    "PprofGoroutine",
			path:    "/debug/pprof/goroutine",
			methods: "GET",
			handler: pprof.Index,
		},
		{
			name:    "PprofHeap",
			path:    "/debug/pprof/heap",
			methods: "GET",
			handler: pprof.Index,
		},
		{
			name:    "PprofMutex",
			path:    "/debug/pprof/mutex",
			methods: "GET",
			handler: pprof.Index,
		},
		{
			name:    "PprofCmdline",
			path:    "/debug/pprof/cmdline",
			methods: "GET",
			handler: pprof.Cmdline,
		},
		{
			name:    "PprofProfile",
			path:    "/debug/pprof/profile",
			methods: "GET",
			handler: pprof.Profile,
		},
		{
			name:    "PprofSymbol",
			path:    "/debug/pprof/symbol",
			methods: "GET",
			handler: pprof.Symbol,
		},
		{
			name:    "PprofTrace",
			path:    "/debug/pprof/trace",
			methods: "GET",
			handler: pprof.Trace,
		},
		{
			name:    "Metrics",
			path:    "/metrics",
			methods: "GET",
			handler: promhttp.Handler().ServeHTTP,
		},
	} {
		if route.excluded {
			continue
		}

		// Register the route
		mux.HandleFunc(route.path, route.handler).Name(route.name).Methods(route.methods)
	}

	n := negroni.New()

	// Panic recovery
	n.Use(negroni.NewRecovery())

	// Use logrus as logger
	n.Use(newLogrusMiddleware(s.log.Logger, s.config.HTTPServer.LogExcludePaths))

	// Allow gzip requests
	n.Use(gzip.Gzip(gzip.DefaultCompression))

	// Add basic auth if configured
	if s.config.HTTPServer.BasicAuth {
		log.Info("server will require basic authentication")
		n.Use(NewBasicAuthMiddleware(s.config.HTTPServer.BasicAuthUser, s.config.HTTPServer.BasicAuthPassword))
	}

	// Add token auth middleware if token configuration file specified
	if s.authManager != nil {
		n.Use(auth.NewMiddleware(s.authManager, mux))
		mux.HandleFunc("/tokens/allowed", s.tokenGetAllowed).Name("TokenGetAllowed")
	}

	// Wrap the router
	n.UseHandler(mux)

	return &http.Server{Addr: addr, Handler: n}
}
