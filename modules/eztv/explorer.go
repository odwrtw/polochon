package eztv

import (
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/eztv"
	polochon "github.com/odwrtw/polochon/lib"
)

// AvailableMovieOptions implements the Explorer interface
func (e *Eztv) AvailableMovieOptions() []polochon.ExplorerOption {
	return []polochon.ExplorerOption{}
}

// AvailableShowOptions implements the Explorer interface
func (e *Eztv) AvailableShowOptions() []polochon.ExplorerOption {
	return []polochon.ExplorerOption{}
}

// GetMovieList implements the Explorer interface
func (e *Eztv) GetMovieList(polochon.ExplorerOption, *logrus.Entry) ([]*polochon.Movie, error) {
	return nil, polochon.ErrNotAvailable
}

// GetShowList implements the Explorer interface
func (e *Eztv) GetShowList(polochon.ExplorerOption, *logrus.Entry) ([]*polochon.Show, error) {
	// Get the page of the shows
	showList, err := eztv.ListShows(1)
	if err != nil {
		return nil, err
	}

	// // Get the details for each show we got
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
