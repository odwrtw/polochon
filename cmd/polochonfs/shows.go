package main

import (
	"context"
	"fmt"

	polochon "github.com/odwrtw/polochon/lib"
	log "github.com/sirupsen/logrus"
)

func (pfs *polochonfs) updateShows(ctx context.Context) {
	dir := pfs.root.getChild(showDirName)

	log.Debug("Fecthing shows")
	shows, err := pfs.client.GetShows()
	if err != nil {
		log.WithField("error", err).Error("Failed to get shows")
		dir.rmAllChilds()
		return
	}

	clear(showInodes)
	dir.rmAllChilds()
	for _, s := range shows.List() {
		showDirNode := newNodeDir(s.Title)
		showDirNode.times = pfs.root.times
		dir.addChild(showDirNode)

		for _, season := range s.Seasons {
			seasonDir := newNodeDir(fmt.Sprintf("Season %d", season.Season))
			seasonDir.times = pfs.root.times
			showDirNode.addChild(seasonDir)

			for _, episode := range season.Episodes {
				url, err := pfs.client.DownloadURL(episode)
				if err != nil {
					log.WithFields(log.Fields{
						"error":   err,
						"show":    s.Title,
						"season":  episode.Season,
						"episode": episode.Episode,
					}).Error("Failed to get episode URL")
					continue
				}

				episodeNode := newNode(episode.Path, url, uint64(episode.Size), episode.DateAdded)
				seasonDir.addChild(episodeNode)

				for _, sub := range episode.Subtitles {
					url, err = pfs.client.DownloadURL(sub)
					if err != nil {
						log.WithFields(log.Fields{
							"error":   err,
							"show":    s.Title,
							"season":  episode.Season,
							"episode": episode.Episode,
							"lang":    sub.Lang,
						}).Error("Failed to get episode subtitle URL")
						continue
					}

					path := polochon.NewFile(episode.Path).SubtitlePath(sub.Lang)
					subNode := newNode(path, url, uint64(sub.Size), episode.DateAdded)
					seasonDir.addChild(subNode)
				}
			}
		}
	}

	log.Debug("Shows updated")
}
