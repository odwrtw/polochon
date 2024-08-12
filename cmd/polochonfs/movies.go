package main

import (
	"context"
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

func (pfs *polochonfs) updateMovies(ctx context.Context) {
	dir := pfs.root.getChild(movieDirName)

	log.Debug("Fecthing movies")
	movies, err := pfs.client.GetMovies()
	if err != nil {
		log.WithField("error", err).Error("Failed to get movies")
		dir.rmAllChilds()
		return
	}

	dir.rmAllChilds()
	for _, m := range movies.List() {
		title := movieDirTitle(m)
		url, err := pfs.client.DownloadURL(m)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"title": m.Title,
			}).Error("Failed to get movie URL")
			continue
		}

		movieDirNode := newNodeDir(title)
		movieDirNode.times = m.DateAdded
		dir.addChild(movieDirNode)

		movieNode := newNode(m.Path, url, uint64(m.Size), m.DateAdded)
		movieDirNode.addChild(movieNode)

		for _, sub := range m.Subtitles {
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
			subNode := newNode(path, url, uint64(sub.Size), m.DateAdded)
			movieDirNode.addChild(subNode)
		}
	}

	log.Debug("Movies updated")
}
