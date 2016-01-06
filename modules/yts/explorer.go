package yts

import (
	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/explorer"
	"github.com/odwrtw/yts"
)

// AvailableShowOptions implements the the explorer interface
func (y *Yts) AvailableShowOptions() []explorer.Option {
	return []explorer.Option{}
}

// GetShowList implements the explorer interface
func (y *Yts) GetShowList(option explorer.Option, log *logrus.Entry) ([]*polochon.Show, error) {
	return nil, polochon.ErrNotAvailable
}

// AvailableMovieOptions implements the the explorer interface
func (y *Yts) AvailableMovieOptions() []explorer.Option {
	return []explorer.Option{
		explorer.BySeeds,
		explorer.ByPeers,
		explorer.ByTitle,
		explorer.ByYear,
		explorer.ByRate,
		explorer.ByDownloadCount,
		explorer.ByLikeCount,
		explorer.ByDateAdded,
	}
}

func translateMovieOptions(expOption explorer.Option) (string, error) {
	translationMap := map[explorer.Option]string{
		explorer.BySeeds:         yts.SortBySeeds,
		explorer.ByPeers:         yts.SortByPeers,
		explorer.ByTitle:         yts.SortByTitle,
		explorer.ByYear:          yts.SortByYear,
		explorer.ByRate:          yts.SortByRating,
		explorer.ByDownloadCount: yts.SortByDownload,
		explorer.ByLikeCount:     yts.SortByLike,
		explorer.ByDateAdded:     yts.SortByDateAdded,
	}
	option, ok := translationMap[expOption]
	if !ok {
		return "", polochon.ErrNotAvailable
	}

	return option, nil
}

// GetMovieList implements the explorer interface
func (y *Yts) GetMovieList(option explorer.Option, log *logrus.Entry) ([]*polochon.Movie, error) {
	log = log.WithField("explore_category", "movies")

	opt, err := translateMovieOptions(option)
	if err != nil {
		return nil, err
	}

	movieList, err := yts.GetList(1, 6, opt, yts.OrderDesc)
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
