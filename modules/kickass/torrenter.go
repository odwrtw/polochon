package kickass

import (
	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/guessit"
	"github.com/odwrtw/kickass"
	"github.com/odwrtw/polochon/lib"
)

// Searcher is an interface to search for torrents
type Searcher interface {
	validate() error
	searchStr() string
	category() Category
	users() []string
	isValidGuess(*guessit.Response, *logrus.Entry) bool
}

// GetTorrents implements the torrenter interface
func (k *Kickass) GetTorrents(i interface{}, log *logrus.Entry) error {
	switch video := i.(type) {
	case *polochon.Movie:
		m := NewMovieSearcher(video, k.MoviesUsers)
		t, err := k.search(m, log)
		if err != nil {
			return err
		}
		m.Torrents = t
		return nil
	case *polochon.ShowEpisode:
		s := NewShowEpisodeSearcher(video, k.ShowsUsers)
		t, err := k.search(s, log)
		if err != nil {
			return err
		}
		s.Torrents = t
		return nil
	default:
		return ErrInvalidType
	}
}

func (k *Kickass) search(s Searcher, log *logrus.Entry) ([]polochon.Torrent, error) {
	users := s.users()
	result := []polochon.Torrent{}

	log = log.WithFields(logrus.Fields{
		"search_category": s.category(),
		"search_string":   s.searchStr(),
	})

	if err := s.validate(); err != nil {
		log.Error(err)
		return nil, err
	}

	for _, u := range users {
		torrents, err := k.searchUser(s, log, u)
		if err != nil {
			return nil, err
		}

		result = append(result, torrents...)
	}

	return polochon.FilterTorrents(result), nil
}

func (k *Kickass) searchUser(s Searcher, log *logrus.Entry, user string) ([]polochon.Torrent, error) {
	query := &kickass.Query{
		User:     user,
		OrderBy:  "seeders",
		Order:    "desc",
		Category: string(s.category()),
		Search:   s.searchStr(),
	}
	log = log.WithField("search_user", user)

	torrents, err := k.client.Search(query)
	if err != nil {
		return nil, err
	}

	result := []polochon.Torrent{}
	for _, t := range torrents {
		torrentStr := torrentGuessitStr(t)
		guess, err := guessit.Guess(torrentStr)
		if err != nil {
			continue
		}

		if !s.isValidGuess(guess, log) {
			continue
		}

		// Default quality
		if s.category() == ShowsCategory && guess.Quality == "" {
			guess.Quality = string(polochon.Quality480p)
		}

		// Get the torrent quality
		torrentQuality := polochon.Quality(guess.Quality)
		if !torrentQuality.IsAllowed() {
			log.Infof("kickass: unhandled quality: %q", torrentQuality)
			continue
		}

		log.WithFields(logrus.Fields{
			"torrent_quality": guess.Quality,
			"torrent_name":    torrentStr,
		}).Debug("Adding torrent to the list")

		// Add the torrent
		result = append(result, polochon.Torrent{
			Quality:    torrentQuality,
			URL:        t.MagnetURL,
			Seeders:    t.Seed,
			Leechers:   t.Leech,
			Source:     moduleName,
			UploadUser: user,
		})
	}

	return result, nil
}
