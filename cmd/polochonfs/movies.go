package main

import (
	"fmt"
	"strings"

	"github.com/odwrtw/polochon/lib/papi"
	log "github.com/sirupsen/logrus"
)

func movieDirTitle(m *papi.Movie) string {
	dirTitle := m.Title
	if m.Year != 0 {
		dirTitle += fmt.Sprintf(" (%d)", m.Year)
	}

	// Replace the "/" in the names with a "-".
	// e.g. 50/50 -> 50-50
	return strings.ReplaceAll(dirTitle, "/", "-")
}

func (pfs *polochonfs) updateMovies() {
	log.Debug("Fecthing movies")
	movies, err := pfs.client.GetMovies()
	if err != nil {
		log.WithField("error", err).Error("Failed to get movies")
		// TODO: should we remove all the files if we can't get an update ?
		return
	}

	movieRootDir := pfs.createDirNode(pfs.root, movieDirName, pfs.root.times)
	movieRootDir.invalidate()
	movieRootDir.valid = true

	for _, m := range movies.List() {
		movieDirNode := pfs.createDirNode(movieRootDir, movieDirTitle(m), m.DateAdded)

		err = pfs.createFileNode(movieDirNode, m, m.Path, m.Size, m.DateAdded)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"title": m.Title,
			}).Error("Failed to create movie node")
			continue
		}

		pfs.createSubtitlesNodes(movieDirNode, m.Path, m.Subtitles, m.DateAdded)
		pfs.createFilesNodes(movieDirNode, []*papi.File{m.Fanart, m.Thumb, m.NFO}, m.DateAdded)
	}

	movieRootDir.clear()

	log.Debug("Movies updated")
}
