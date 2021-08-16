package main

import (
	"fmt"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/papi"
)

func (fs *fs) handleMovies() error {
	moviesNode := fs.root.addDir("movies", time.Now())

	moviesCol, err := fs.client.GetMovies()
	if err != nil {
		return err
	}

	for _, m := range moviesCol.List() {
		if err := fs.addMovie(moviesNode, m); err != nil {
			return err
		}
	}

	return nil
}

func (fs *fs) addMovie(root *node, m *papi.Movie) error {
	dirTitle := m.Title
	if m.Year != 0 {
		dirTitle += fmt.Sprintf(" (%d)", m.Year)
	}

	dir := root.addDir(dirTitle, m.DateAdded)

	url, err := fs.client.DownloadURL(m)
	if err != nil {
		return err
	}

	dir.add(m.Path, url, m.DateAdded, m.Size)

	for _, sub := range m.Subtitles {
		url, err = fs.client.DownloadURL(sub)
		if err != nil {
			return err
		}

		path := polochon.NewFile(m.Path).SubtitlePath(sub.Lang)

		dir.add(path, url, m.DateAdded, sub.Size)
	}

	return nil
}
