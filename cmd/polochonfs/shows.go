package main

import (
	"context"
	"fmt"

	polochon "github.com/odwrtw/polochon/lib"
)

func (pfs *polochonfs) updateShows(ctx context.Context) {
	dir := pfs.root.getChild("shows")

	for _, s := range pfs.shows {
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
					fmt.Println("Failed to get episode URL:", err)
					continue
				}

				episodeNode := newNode(episode.Path, url, uint64(episode.Size), episode.DateAdded)
				seasonDir.addChild(episodeNode)

				for _, sub := range episode.Subtitles {
					url, err = pfs.client.DownloadURL(sub)
					if err != nil {
						fmt.Println("Failed to get episode sub URL:", err)
						continue
					}

					path := polochon.NewFile(episode.Path).SubtitlePath(sub.Lang)
					subNode := newNode(path, url, uint64(sub.Size), episode.DateAdded)
					seasonDir.addChild(subNode)
				}
			}
		}
	}
}
