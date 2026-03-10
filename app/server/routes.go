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
			path:    "/movies",
			methods: "GET",
			handler: s.movieIndex,
		},
		{
			path:    "/movies/{id}",
			methods: "GET",
			handler: s.getMovieDetails,
		},
		{
			path:    "/movies/{id}",
			methods: "DELETE",
			handler: s.deleteMovie,
		},
		{
			path:     "/movies/{id}/download",
			methods:  "GET",
			handler:  s.serveMovie,
			excluded: !s.config.HTTPServer.ServeFiles,
		},
		{
			path:     "/movies/{id}/download/{filename}",
			methods:  "GET",
			handler:  s.serveMovie,
			excluded: !s.config.HTTPServer.ServeFiles,
		},
		{
			path:     "/movies/{id}/files/{name}",
			methods:  "GET",
			handler:  s.serveMovieFile,
			excluded: !s.config.HTTPServer.ServeFiles,
		},
		{
			path:     "/movies/{id}/subtitles/{lang}/download",
			methods:  "GET",
			handler:  s.serveMovieSubtitle,
			excluded: !s.config.HTTPServer.ServeFiles,
		},
		{
			path:     "/movies/{id}/subtitles/{lang}/download/{filename}",
			methods:  "GET",
			handler:  s.serveMovieSubtitle,
			excluded: !s.config.HTTPServer.ServeFiles,
		},
		{
			path:    "/movies/{id}/subtitles/{lang}",
			methods: "POST",
			handler: s.updateMovieSubtitle,
		},
		{
			path:    "/movies/{id}/subtitles/{lang}",
			methods: "PUT",
			handler: s.uploadMovieSubtitle,
		},
		{
			path:    "/shows",
			methods: "GET",
			handler: s.showIds,
		},
		{
			path:    "/shows/{id}",
			methods: "GET",
			handler: s.getShowDetails,
		},
		{
			path:    "/shows/{id}",
			methods: "DELETE",
			handler: s.deleteShow,
		},
		{
			path:    "/shows/{id}/files/{name}",
			methods: "GET",
			handler: s.getShowFiles,
		},
		{
			path:    "/shows/{id}/seasons/{season:[0-9]+}",
			methods: "GET",
			handler: s.getSeasonDetails,
		},
		{
			path:    "/shows/{id}/seasons/{season:[0-9]+}",
			methods: "DELETE",
			handler: s.deleteSeason,
		},
		{
			path:    "/shows/{id}/seasons/{season:[0-9]+}/episodes/{episode:[0-9]+}",
			methods: "GET",
			handler: s.getShowEpisodeIDDetails,
		},
		{
			path:    "/shows/{id}/seasons/{season:[0-9]+}/episodes/{episode:[0-9]+}",
			methods: "DELETE",
			handler: s.deleteEpisode,
		},
		{
			path:    "/shows/{id}/seasons/{season:[0-9]+}/episodes/{episode:[0-9]+}/subtitles/{lang}",
			methods: "POST",
			handler: s.updateEpisodeSubtitle,
		},
		{
			path:    "/shows/{id}/seasons/{season:[0-9]+}/episodes/{episode:[0-9]+}/subtitles/{lang}",
			methods: "PUT",
			handler: s.uploadEpisodeSubtitle,
		},
		{
			path:     "/shows/{id}/seasons/{season:[0-9]+}/episodes/{episode:[0-9]+}/download",
			methods:  "GET",
			handler:  s.serveEpisode,
			excluded: !s.config.HTTPServer.ServeFiles,
		},
		{
			path:     "/shows/{id}/seasons/{season:[0-9]+}/episodes/{episode:[0-9]+}/download/{filename}",
			methods:  "GET",
			handler:  s.serveEpisode,
			excluded: !s.config.HTTPServer.ServeFiles,
		},
		{
			path:    "/shows/{id}/seasons/{season:[0-9]+}/episodes/{episode:[0-9]+}/files/{name}",
			methods: "GET",
			handler: s.getShowEpisodeFiles,
		},
		{
			path:     "/shows/{id}/seasons/{season:[0-9]+}/episodes/{episode:[0-9]+}/subtitles/{lang}/download",
			methods:  "GET",
			handler:  s.serveEpisodeSubtitle,
			excluded: !s.config.HTTPServer.ServeFiles,
		},
		{
			path:     "/shows/{id}/seasons/{season:[0-9]+}/episodes/{episode:[0-9]+}/subtitles/{lang}/download/{filename}",
			methods:  "GET",
			handler:  s.serveEpisodeSubtitle,
			excluded: !s.config.HTTPServer.ServeFiles,
		},
		{
			path:    "/events",
			methods: "GET",
			handler: s.events,
		},
		{
			path:    "/wishlist",
			methods: "GET",
			handler: s.wishlist,
		},
		{
			path:    "/torrents",
			methods: "POST",
			handler: s.addTorrent,
		},
		{
			path:    "/torrents",
			methods: "GET",
			handler: s.getTorrents,
		},
		{
			path:    "/torrents/{id}",
			methods: "DELETE",
			handler: s.removeTorrent,
		},
		{
			path:    "/library/refresh",
			methods: "POST",
			handler: s.libraryRefresh,
		},
		{
			path:    "/modules/status",
			methods: "GET",
			handler: s.getModulesStatus,
		},
		{
			path:    "/debug/pprof/",
			methods: "GET",
			handler: pprof.Index,
		},
		{
			path:    "/debug/pprof/block",
			methods: "GET",
			handler: pprof.Index,
		},
		{
			path:    "/debug/pprof/goroutine",
			methods: "GET",
			handler: pprof.Index,
		},
		{
			path:    "/debug/pprof/heap",
			methods: "GET",
			handler: pprof.Index,
		},
		{
			path:    "/debug/pprof/mutex",
			methods: "GET",
			handler: pprof.Index,
		},
		{
			path:    "/debug/pprof/cmdline",
			methods: "GET",
			handler: pprof.Cmdline,
		},
		{
			path:    "/debug/pprof/profile",
			methods: "GET",
			handler: pprof.Profile,
		},
		{
			path:    "/debug/pprof/symbol",
			methods: "GET",
			handler: pprof.Symbol,
		},
		{
			path:    "/debug/pprof/trace",
			methods: "GET",
			handler: pprof.Trace,
		},
		{
			path:    "/metrics",
			methods: "GET",
			handler: promhttp.Handler().ServeHTTP,
		},
	} {
		if route.excluded {
			continue
		}

		// Register the route
		mux.HandleFunc(route.path, route.handler).Methods(route.methods)
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
		n.Use(auth.NewMiddleware(s.authManager))
	}

	// Wrap the router
	n.UseHandler(mux)

	return &http.Server{Addr: addr, Handler: n}
}
