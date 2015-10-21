package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/meatballhat/negroni-logrus"
	"github.com/odwrtw/polochon/lib"
	"github.com/phyber/negroni-gzip/gzip"
)

func (a *App) movieSlugs(w http.ResponseWriter, req *http.Request) {
	a.logger.Debug("Listing movies by slugs")

	movieSlugs, err := a.videoStore.MovieSlugs()
	if err != nil {
		a.render.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	a.render.JSON(w, http.StatusOK, movieSlugs)
}

func (a *App) movieIds(w http.ResponseWriter, req *http.Request) {
	a.logger.Debug("Listing movies by ids")

	movieIds, err := a.videoStore.MovieIds()
	if err != nil {
		a.render.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	a.render.JSON(w, http.StatusOK, movieIds)
}

func (a *App) showIds(w http.ResponseWriter, req *http.Request) {
	a.logger.Debug("Listing shows")

	ids, err := a.videoStore.ShowIds()
	if err != nil {
		a.render.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
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

	a.render.JSON(w, http.StatusOK, ret)
}

func (a *App) showSlugs(w http.ResponseWriter, req *http.Request) {
	a.logger.Debug("Listing shows by slugs")

	slugs, err := a.videoStore.ShowSlugs()
	if err != nil {
		a.render.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	a.render.JSON(w, http.StatusOK, slugs)
}

func (a *App) wishlist(w http.ResponseWriter, req *http.Request) {
	wl := polochon.NewWishlist(a.config.Wishlist, logrus.NewEntry(a.logger))

	if err := wl.Fetch(); err != nil {
		a.render.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	a.render.JSON(w, http.StatusOK, wl)
}

func serveFile(w http.ResponseWriter, r *http.Request, file *polochon.File) {
	// Set the header so that when downloading, the real filename will be given
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(file.Path)))
	http.ServeFile(w, r, file.Path)
}

func (a *App) serveMovie(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	var err error
	var v polochon.Video

	if vars["slug"] != "" {
		v, err = a.videoStore.SearchMovieBySlug(vars["slug"])
	} else if vars["id"] != "" {
		v, err = a.videoStore.SearchMovieByImdbID(vars["id"])
	} else {
		a.render.JSON(w, http.StatusNotFound, map[string]string{"error": "URL not found"})
		return
	}

	if err != nil {
		a.logger.Error(err)
		var status int
		if err == polochon.ErrSlugNotFound {
			status = http.StatusNotFound
		} else {
			status = http.StatusInternalServerError
		}
		a.render.JSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	serveFile(w, req, v.GetFile())
}

func (a *App) serveShow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	var err error
	var v polochon.Video

	if vars["slug"] != "" {
		v, err = a.videoStore.SearchShowEpisodeBySlug(vars["slug"])
	} else if vars["id"] != "" && vars["season"] != "" && vars["episode"] != "" {
		sStr := vars["season"]
		eStr := vars["episode"]
		season, err := strconv.Atoi(sStr)
		episode, err := strconv.Atoi(eStr)
		if err != nil {
			a.render.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		v, err = a.videoStore.SearchShowEpisodeByImdbID(vars["id"], season, episode)
	} else {
		a.render.JSON(w, http.StatusNotFound, map[string]string{"error": "URL not found"})
		return
	}

	if err != nil {
		a.logger.Error(err)
		var status int
		if err == polochon.ErrSlugNotFound {
			status = http.StatusNotFound
		} else {
			status = http.StatusInternalServerError
		}
		a.render.JSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	serveFile(w, r, v.GetFile())

}

func (a *App) deleteFile(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	videoType := vars["videoType"]
	slug := vars["slug"]

	a.logger.Debugf("Looking for the %s: %s", videoType, slug)

	var searchFunc func(slug string) (polochon.Video, error)
	switch videoType {
	case "movies":
		searchFunc = a.videoStore.SearchMovieBySlug
	case "shows":
		searchFunc = a.videoStore.SearchShowEpisodeBySlug
	default:
		msg := fmt.Sprintf("Invalid video type: %q", videoType)
		a.render.JSON(w, http.StatusInternalServerError, map[string]string{"error": msg})
		return
	}

	// Find the file by Slug
	v, err := searchFunc(slug)
	if err != nil {
		a.logger.Error(err)
		var status int
		if err == polochon.ErrSlugNotFound {
			status = http.StatusNotFound
		} else {
			status = http.StatusInternalServerError
		}
		a.render.JSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	videoFile := v.GetFile()
	a.logger.Debugf("Got the file to delete: %s", filepath.Base(videoFile.Path))

	err = a.videoStore.Delete(v)
	if err != nil {
		a.render.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	a.render.JSON(w, http.StatusOK, nil)
}

func (a *App) addTorrent(w http.ResponseWriter, r *http.Request) {
	if !a.config.Downloader.Enabled {
		a.logger.Warning("Downloader not available")
		http.Error(w, "Downloader not enabled in your polochon", http.StatusServiceUnavailable)
		return
	}

	req := new(struct{ URL string })
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		http.Error(w, "Unkown error", http.StatusInternalServerError)
		a.logger.Warning(err.Error())
		return
	}
	if req.URL == "" {
		http.Error(w, "URL missing", http.StatusBadRequest)
		return
	}

	if err := a.config.Downloader.Client.Download(req.URL, a.logger.WithField("function", "downloader")); err != nil {
		if err == polochon.ErrDuplicateTorrent {
			http.Error(w, "Torrent already added", http.StatusConflict)
			return
		}
		a.logger.Warning("Error while adding a torrent via the API: %q", err)
		http.Error(w, "Unkown error", http.StatusInternalServerError)
		return
	}
}

// HTTPServer handles the HTTP requests
func (a *App) HTTPServer() {
	addr := fmt.Sprintf("%s:%d", a.config.HTTPServer.Host, a.config.HTTPServer.Port)
	a.logger.Debugf("HTTP Server listening on: %s", addr)

	a.mux.HandleFunc("/movies/slugs", a.movieSlugs)
	a.mux.HandleFunc("/movies/ids", a.movieIds)
	a.mux.HandleFunc("/shows/ids", a.showIds)
	a.mux.HandleFunc("/shows/slugs", a.showSlugs)
	a.mux.HandleFunc("/wishlist", a.wishlist)

	a.mux.HandleFunc("/torrents", a.addTorrent).Methods("POST")

	if a.config.HTTPServer.ServeFiles {
		a.logger.Info("Server is serving files")
		a.mux.HandleFunc("/{videoType:movies|shows}/slugs/{slug}/delete", a.deleteFile)
		a.mux.HandleFunc("/shows/slugs/{slug}/download", a.serveShow)
		a.mux.HandleFunc("/shows/ids/{id}/{season}/{episode}/download", a.serveShow)
		a.mux.HandleFunc("/movies/ids/{id}/download", a.serveMovie)
		a.mux.HandleFunc("/movies/slugs/{slug}/download", a.serveMovie)
	}

	n := negroni.New()
	// Panic recovery
	n.Use(negroni.NewRecovery())
	// Use logrus as logger
	n.Use(negronilogrus.NewCustomMiddleware(logrus.InfoLevel, a.logger.Formatter, "httpServer"))
	// gzip compression
	n.Use(gzip.Gzip(gzip.DefaultCompression))

	// Add basic auth if configured
	if a.config.HTTPServer.BasicAuth {
		a.logger.Info("Server requires basic authentication")
		n.Use(NewBasicAuthMiddleware(a.config.HTTPServer.BasicAuthUser, a.config.HTTPServer.BasicAuthPassword))
	}

	// Wrap the router
	n.UseHandler(a.mux)

	// Serve HTTP
	if err := http.ListenAndServe(addr, n); err != nil {
		a.logger.Error("Couldn't start the HTTP server : ", err)
		a.Stop()
	}
}
