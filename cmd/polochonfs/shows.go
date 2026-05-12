package main

import (
	"fmt"
	"log/slog"

	"github.com/odwrtw/polochon/lib/papi"
)

func (pfs *polochonfs) updateShows() {
	slog.Debug("Fecthing shows")
	shows, err := pfs.client.GetShows()
	if err != nil {
		slog.Error("Failed to get shows", "error", err)
		// TODO: remove all files ?
		return
	}

	showRootDir := pfs.createDirNode(pfs.root, showDirName, pfs.root.times)
	showRootDir.invalidate()
	showRootDir.valid = true

	for _, s := range shows.List() {
		showDirNode := pfs.createDirNode(showRootDir, s.Title, pfs.root.times)

		files := s.SidecarFiles()
		pfs.createFilesNodes(showDirNode, files, showDirNode.times)

		for _, season := range s.Seasons {
			name := fmt.Sprintf("Season %d", season.Season)
			seasonDir := pfs.createDirNode(showDirNode, name, showDirNode.times)

			for _, episode := range season.Episodes {
				err = pfs.createFileNode(seasonDir, episode, episode.Path, episode.Size, episode.DateAdded)
				if err != nil {
					slog.Error("Failed to create episode node",
						"error", err,
						"show", s.Title,
						"season", episode.Season,
						"episode", episode.Episode,
					)
					continue
				}

				pfs.createFilesNodes(seasonDir, []*papi.File{episode.SidecarFile()}, episode.DateAdded)
				pfs.createSubtitlesNodes(seasonDir, episode.Path, episode.Subtitles, episode.DateAdded)
			}
		}
	}

	showRootDir.clear()

	slog.Debug("Shows updated")
}
