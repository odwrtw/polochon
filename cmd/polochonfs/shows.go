package main

import (
	"fmt"

	"github.com/odwrtw/polochon/lib/papi"
	log "github.com/sirupsen/logrus"
)

func (pfs *polochonfs) updateShows() {
	log.Debug("Fecthing shows")
	shows, err := pfs.client.GetShows()
	if err != nil {
		log.WithField("error", err).Error("Failed to get shows")
		// TODO: remove all files ?
		return
	}

	showRootDir := pfs.createDirNode(pfs.root, showDirName, pfs.root.times)
	showRootDir.invalidate()
	showRootDir.valid = true

	for _, s := range shows.List() {
		showDirNode := pfs.createDirNode(showRootDir, s.Title, pfs.root.times)

		files := []*papi.File{s.Fanart, s.Banner, s.Poster, s.NFO}
		pfs.createFilesNodes(showDirNode, files, showDirNode.times)

		for _, season := range s.Seasons {
			name := fmt.Sprintf("Season %d", season.Season)
			seasonDir := pfs.createDirNode(showDirNode, name, showDirNode.times)

			for _, episode := range season.Episodes {
				err = pfs.createFileNode(seasonDir, episode, episode.Path, episode.Size, episode.DateAdded)
				if err != nil {
					log.WithFields(log.Fields{
						"error":   err,
						"show":    s.Title,
						"season":  episode.Season,
						"episode": episode.Episode,
					}).Error("Failed to create episode node")
					continue
				}

				files := []*papi.File{episode.NFO}
				pfs.createFilesNodes(seasonDir, files, episode.DateAdded)
				pfs.createSubtitlesNodes(seasonDir, episode.Path, episode.Subtitles, episode.DateAdded)
			}
		}
	}

	showRootDir.clear()

	log.Debug("Shows updated")
}
