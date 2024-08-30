package main

import (
	"fmt"
	"strings"

	polochon "github.com/odwrtw/polochon/lib"
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
	dir := pfs.root.getChild(movieDirName)

	log.Debug("Fecthing movies")
	movies, err := pfs.client.GetMovies()
	if err != nil {
		log.WithField("error", err).Error("Failed to get movies")
		dir.rmAllChilds()
		return
	}

	clear(movieInodes)
	dir.rmAllChilds()
	for _, m := range movies.List() {
		imdbID := m.ImdbID
		title := movieDirTitle(m)
		url, err := pfs.client.DownloadURL(m)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"title": m.Title,
			}).Error("Failed to get movie URL")
			continue
		}

		movieDirNode := newNodeDir(imdbID, title)
		movieDirNode.times = m.DateAdded
		dir.addChild(movieDirNode)

		movieNode := newNode(imdbID, m.Path, url, uint64(m.Size), m.DateAdded)
		movieDirNode.addChild(movieNode)

		for _, sub := range m.Subtitles {
			if sub.Embedded {
				continue
			}

			url, err := pfs.client.DownloadURL(sub)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
					"title": m.Title,
					"lang":  sub.Lang,
				}).Error("Failed to get movie subtitle URL")
				continue
			}

			path := polochon.NewFile(m.Path).SubtitlePath(sub.Lang)
			subNode := newNode(imdbID, path, url, uint64(sub.Size), m.DateAdded)
			movieDirNode.addChild(subNode)
		}

		for _, file := range []*papi.File{m.Fanart, m.Thumb, m.NFO} {
			if file == nil {
				continue
			}

			url, err := pfs.client.DownloadURL(file)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
					"title": m.Title,
				}).Error("Failed to get movie meta URL")
				continue
			}

			fileNode := newNode(
				imdbID, file.Name, url,
				uint64(file.Size), m.DateAdded)
			movieDirNode.addChild(fileNode)
		}
	}

	log.Debug("Movies updated")
}
