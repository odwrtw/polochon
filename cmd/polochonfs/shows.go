package main

import (
	"fmt"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/papi"
	log "github.com/sirupsen/logrus"
)

func (pfs *polochonfs) updateShows() {
	dir := pfs.root.getChild(showDirName)

	defer func() {
		pfs.root.addChild(dir)
		_ = pfs.root.NotifyEntry(showDirName)
	}()

	log.Debug("Fecthing shows")
	shows, err := pfs.client.GetShows()
	if err != nil {
		log.WithField("error", err).Error("Failed to get shows")
		dir.rmAllChildren()
		return
	}

	clear(showInodes)
	dir.rmAllChildren()
	for _, s := range shows.List() {
		imdbID := s.ImdbID
		showDirNode := newNodeDir(imdbID, s.Title, pfs.root.times)
		dir.addChild(showDirNode)

		for _, file := range []*papi.File{s.Fanart, s.Banner, s.Poster, s.NFO} {
			if file == nil {
				continue
			}

			url, err := pfs.client.URI(file)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
					"title": s.Title,
				}).Error("Failed to get file URL")
				continue
			}

			fileNode := newNode(imdbID, file.Name, url,
				uint64(file.Size), showDirNode.times)
			showDirNode.addChild(fileNode)
		}

		for _, season := range s.Seasons {
			seasonDir := newNodeDir(season.ShowImdbID,
				fmt.Sprintf("Season %d", season.Season),
				pfs.root.times)
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

				episodeNode := newNode(
					imdbID, episode.Path, url,
					uint64(episode.Size), episode.DateAdded)
				seasonDir.addChild(episodeNode)

				if episode.NFO != nil {
					uri, err := pfs.client.URI(episode.NFO)
					if err != nil {
						log.WithFields(log.Fields{
							"error":   err,
							"show":    s.Title,
							"season":  episode.Season,
							"episode": episode.Episode,
						}).Error("Failed to get NFO URL")
					}

					fileNode := newNode(
						imdbID, episode.NFO.Name, uri,
						uint64(episode.NFO.Size), episode.DateAdded)
					seasonDir.addChild(fileNode)
				}

				for _, sub := range episode.Subtitles {
					if sub.Embedded {
						continue
					}

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
					subNode := newNode(
						imdbID, path, url,
						uint64(sub.Size), episode.DateAdded)
					seasonDir.addChild(subNode)
				}
			}
		}
	}

	log.Debug("Shows updated")
}
