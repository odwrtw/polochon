package yts

import (
	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/yts"
)

// NewExplorer returns a new explorer
func NewExplorer(p []byte) (polochon.Explorer, error) {
	return &Yts{}, nil
}

// AvailableShowOptions implements the the explorer interface
func (y *Yts) AvailableShowOptions() []string {
	return []string{}
}

// GetShowList implements the explorer interface
func (y *Yts) GetShowList(option string, log *logrus.Entry) ([]*polochon.Show, error) {
	return nil, polochon.ErrNotAvailable
}

// AvailableMovieOptions implements the the explorer interface
func (y *Yts) AvailableMovieOptions() []string {
	return []string{
		yts.SortBySeeds,
		yts.SortByPeers,
		yts.SortByTitle,
		yts.SortByYear,
		yts.SortByRating,
		yts.SortByDownload,
		yts.SortByLike,
		yts.SortByDateAdded,
	}
}

// GetMovieList implements the explorer interface
func (y *Yts) GetMovieList(option string, log *logrus.Entry) ([]*polochon.Movie, error) {
	log = log.WithField("explore_category", "movies")

	movieList, err := yts.GetList(1, 6, option, yts.OrderDesc)
	if err != nil {
		return nil, err
	}

	result := []*polochon.Movie{}
	for _, movie := range movieList {
		// Create the movie
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
