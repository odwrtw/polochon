package yts

import (
	"fmt"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/yts"
	"github.com/sirupsen/logrus"
)

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
		m.Torrents = polochonTorrents(&movie)

		result = append(result, m)
	}
	return result, nil
}

// SearchShow implements polochon Searcher interface
func (y *Yts) SearchShow(key string, log *logrus.Entry) ([]*polochon.Show, error) {
	return nil, fmt.Errorf("yts: not implemented")
}
