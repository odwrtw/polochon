package kickass

import (
	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/guessit"
	"github.com/odwrtw/kickass"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/explorer"
)

// AvailableShowOptions implements the the explorer interface
func (k *Kickass) AvailableShowOptions() []explorer.Option {
	return []explorer.Option{}
}

// GetShowList implements the explorer interface
func (k *Kickass) GetShowList(option explorer.Option, log *logrus.Entry) ([]*polochon.Show, error) {
	return nil, polochon.ErrNotAvailable
}

// AvailableMovieOptions implements the the explorer interface
func (k *Kickass) AvailableMovieOptions() []explorer.Option {
	return []explorer.Option{explorer.BySeeds}
}

// GetMovieList implements the explorer interface
func (k *Kickass) GetMovieList(option explorer.Option, log *logrus.Entry) ([]*polochon.Movie, error) {
	log = log.WithField("explore_category", "movies")

	// Only the seeders option is available for this module
	if option != explorer.BySeeds {
		return nil, polochon.ErrNotAvailable
	}

	movies := map[string]*polochon.Movie{}
	for _, u := range k.MoviesUsers {
		if err := k.listMoviesByUser(movies, u, log); err != nil {
			return nil, err
		}
	}

	result := []*polochon.Movie{}
	for _, m := range movies {
		result = append(result, m)
	}

	return result, nil
}

func (k *Kickass) listMoviesByUser(movies map[string]*polochon.Movie, user string, log *logrus.Entry) error {
	query := &kickass.Query{
		User:     user,
		OrderBy:  "seeders",
		Order:    "desc",
		Category: string(MoviesCategory),
	}
	log = log.WithField("explore_user", user)

	torrents, err := k.client.ListByUser(query)
	if err != nil {
		return err
	}

	for _, t := range torrents {
		torrentStr := torrentGuessitStr(t)
		guess, err := guessit.Guess(torrentStr)
		if err != nil {
			continue
		}

		// Get the torrent quality
		torrentQuality := polochon.Quality(guess.Quality)
		if !torrentQuality.IsAllowed() {
			log.Infof("kickass: unhandled quality: %q", torrentQuality)
			continue
		}

		// Get the movie if its already in the map
		m, ok := movies[guess.Title]
		if !ok {
			// Create a new movie
			m = polochon.NewMovie(polochon.MovieConfig{})
			m.Title = guess.Title
			if guess.Year != 0 {
				m.Year = guess.Year
			}
		}

		log.WithFields(logrus.Fields{
			"torrent_quality": guess.Quality,
			"movie_title":     guess.Title,
		}).Debug("Adding torrent to the list")

		m.Torrents = append(m.Torrents, polochon.Torrent{
			Quality:    torrentQuality,
			URL:        t.MagnetURL,
			Seeders:    t.Seed,
			Leechers:   t.Leech,
			Source:     moduleName,
			UploadUser: user,
		})

		movies[m.Title] = m
	}

	return nil
}
