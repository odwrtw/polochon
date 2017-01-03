package yts

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/yts"
)

// NewSearcher returns a new searcher
func NewSearcher() (polochon.Searcher, error) {
	return &Yts{}, nil
}

// SearchMovie implements polochon Searcher interface
func (y *Yts) SearchMovie(key string, log *logrus.Entry) ([]*polochon.Movie, error) {
	movieList, err := yts.Search(key)
	if err != nil {
		return nil, err
	}

	result := []*polochon.Movie{}
	for _, movie := range movieList {
		m := polochon.NewMovie(polochon.MovieConfig{})
		m.Title = movie.Title
		m.Year = movie.Year
		m.ImdbID = movie.ImdbID

		// Add the torrents
		for _, t := range movie.Torrents {
			// Get the torrent quality
			torrentQuality := polochon.Quality(t.Quality)
			if !torrentQuality.IsAllowed() {
				log.Debugf("yts: unhandled quality: %q", torrentQuality)
				continue
			}
			m.Torrents = append(m.Torrents, polochon.Torrent{
				Quality:  torrentQuality,
				URL:      t.URL,
				Seeders:  t.Seeds,
				Leechers: t.Peers,
				Source:   moduleName,
			})
		}

		// Append the movie
		result = append(result, m)
	}
	return result, nil
}

// SearchShow implements polochon Searcher interface
func (y *Yts) SearchShow(key string, log *logrus.Entry) ([]*polochon.Show, error) {
	return nil, fmt.Errorf("Not implemented")
}
