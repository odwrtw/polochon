package yts

import (
	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/yts"
)

// NewExplorer returns a new explorer
func NewExplorer() (polochon.Explorer, error) {
	return &Yts{}, nil
}

// AvailableShowOptions implements the the explorer interface
func (y *Yts) AvailableShowOptions() []polochon.ExplorerOption {
	return []polochon.ExplorerOption{}
}

// GetShowList implements the explorer interface
func (y *Yts) GetShowList(option polochon.ExplorerOption, log *logrus.Entry) ([]*polochon.Show, error) {
	return nil, polochon.ErrNotAvailable
}

// AvailableMovieOptions implements the the explorer interface
func (y *Yts) AvailableMovieOptions() []polochon.ExplorerOption {
	return []polochon.ExplorerOption{
		polochon.ExploreBySeeds,
		polochon.ExploreByPeers,
		polochon.ExploreByTitle,
		polochon.ExploreByYear,
		polochon.ExploreByRate,
		polochon.ExploreByDownloadCount,
		polochon.ExploreByLikeCount,
		polochon.ExploreByDateAdded,
	}
}

func translateMovieOptions(expOption polochon.ExplorerOption) (string, error) {
	translationMap := map[polochon.ExplorerOption]string{
		polochon.ExploreBySeeds:         yts.SortBySeeds,
		polochon.ExploreByPeers:         yts.SortByPeers,
		polochon.ExploreByTitle:         yts.SortByTitle,
		polochon.ExploreByYear:          yts.SortByYear,
		polochon.ExploreByRate:          yts.SortByRating,
		polochon.ExploreByDownloadCount: yts.SortByDownload,
		polochon.ExploreByLikeCount:     yts.SortByLike,
		polochon.ExploreByDateAdded:     yts.SortByDateAdded,
	}
	option, ok := translationMap[expOption]
	if !ok {
		return "", polochon.ErrNotAvailable
	}

	return option, nil
}

// GetMovieList implements the explorer interface
func (y *Yts) GetMovieList(option polochon.ExplorerOption, log *logrus.Entry) ([]*polochon.Movie, error) {
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
