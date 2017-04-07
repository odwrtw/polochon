package eztv

import (
	"fmt"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/eztv"
	"github.com/odwrtw/polochon/lib"
)

// Register eztv as a Searcher
func init() {
	polochon.RegisterSearcher(moduleName, NewSearcher)
}

// NewSearcher returns a new searcher
func NewSearcher(p []byte) (polochon.Searcher, error) {
	return &Eztv{}, nil
}

// SearchShow implements polochon Searcher interface
func (e *Eztv) SearchShow(key string, log *logrus.Entry) ([]*polochon.Show, error) {
	showList, err := eztv.SearchShow(key)
	if err != nil {
		return nil, err
	}

	result := []*polochon.Show{}
	for _, show := range showList {
		// Create the movie
		tvdb, err := strconv.Atoi(show.TvdbID)
		if err != nil {
			tvdb = 0
		}
		year, err := strconv.Atoi(show.Year)
		if err != nil {
			year = 0
		}
		s := polochon.NewShow(polochon.ShowConfig{})
		s.ImdbID = show.ImdbID
		s.TvdbID = tvdb
		s.Title = show.Title
		s.Plot = show.Synopsis
		s.Year = year
		s.Banner = show.Images.Banner
		s.Poster = show.Images.Poster
		s.Fanart = show.Images.Fanart

		// Append the show
		result = append(result, s)
	}
	return result, nil
}

// SearchMovie implements polochon Searcher interface
func (e *Eztv) SearchMovie(key string, log *logrus.Entry) ([]*polochon.Movie, error) {
	return nil, fmt.Errorf("Not implemented")
}
