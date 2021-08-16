package main

import (
	"context"
	"fmt"
	"strings"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/papi"
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
	dir := pfs.root.getChild("movies")

	for _, m := range pfs.movies {
		title := movieDirTitle(m)
		url, err := pfs.client.DownloadURL(m)
		if err != nil {
			fmt.Printf("Failed to get url for %s\n", m.Title)
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
				fmt.Printf("Failed to get url for %s sub:%s\n", m.Title, sub.Lang)
				continue
			}

			path := polochon.NewFile(m.Path).SubtitlePath(sub.Lang)
			subNode := newNode(path, url, uint64(sub.Size), m.DateAdded)
			movieDirNode.addChild(subNode)
		}
	}
}
