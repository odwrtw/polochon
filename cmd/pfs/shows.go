package main

import (
	"fmt"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/papi"
)

func (fs *fs) handleShows() error {
	showsNode := fs.root.addDir("shows", time.Now())

	shows, err := fs.client.GetShows()
	if err != nil {
		return err
	}

	for _, s := range shows.List() {
		if err := fs.addShow(showsNode, s); err != nil {
			return err
		}
	}

	return nil
}

func (fs *fs) addShow(root *node, s *papi.Show) error {
	dir := root.addDir(s.Title, time.Now())

	for _, season := range s.Seasons {
		seasonDir := dir.addDir(fmt.Sprintf("Season %d", season.Season), time.Now())

		for _, episode := range season.Episodes {
			url, err := fs.client.DownloadURL(episode)
			if err != nil {
				return err
			}

			seasonDir.add(episode.Path, url, episode.DateAdded, episode.Size)

			for _, sub := range episode.Subtitles {
				url, err = fs.client.DownloadURL(sub)
				if err != nil {
					return err
				}

				path := polochon.NewFile(episode.Path).SubtitlePath(sub.Lang)

				seasonDir.add(path, url, episode.DateAdded, sub.Size)
			}
		}
	}

	return nil
}
