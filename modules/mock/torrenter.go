package mock

import (
	"fmt"
	"math/rand"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// GetTorrents implements the torrenter interface
func (mock *Mock) GetTorrents(i interface{}, log *logrus.Entry) error {
	switch v := i.(type) {
	case *polochon.ShowEpisode:
		mock.getShowEpisodeTorrents(v)
	case *polochon.Movie:
		mock.getMovieTorrents(v)
	default:
		return ErrInvalidArgument
	}
	return nil
}

func (mock *Mock) getShowEpisodeTorrents(e *polochon.ShowEpisode) {
	torrents := []polochon.Torrent{}
	for _, q := range []polochon.Quality{polochon.Quality480p, polochon.Quality720p} {
		torrents = append(torrents, polochon.Torrent{
			URL:      fmt.Sprintf("https://mock.com/t%s.torrent", q),
			Quality:  q,
			Source:   moduleName,
			Seeders:  rand.Intn(100),
			Leechers: rand.Intn(500),
		})
	}

	e.Torrents = torrents
	return
}

func (mock *Mock) getMovieTorrents(m *polochon.Movie) {
	torrents := []polochon.Torrent{}
	for _, q := range []polochon.Quality{polochon.Quality720p, polochon.Quality1080p, polochon.Quality3D} {
		torrents = append(torrents, polochon.Torrent{
			URL:      fmt.Sprintf("https://mock.com/t%s.torrent", q),
			Quality:  q,
			Source:   moduleName,
			Seeders:  rand.Intn(100),
			Leechers: rand.Intn(500),
		})
	}

	m.Torrents = torrents
	return
}

// SearchTorrents implements the torrenter interface
func (mock *Mock) SearchTorrents(string) ([]*polochon.Torrent, error) {
	torrents := []*polochon.Torrent{}
	for _, q := range []polochon.Quality{polochon.Quality720p, polochon.Quality1080p, polochon.Quality3D} {
		torrents = append(torrents, &polochon.Torrent{
			URL:      fmt.Sprintf("https://mock.com/t%s.torrent", q),
			Quality:  q,
			Source:   moduleName,
			Seeders:  rand.Intn(100),
			Leechers: rand.Intn(500),
		})
	}
	return torrents, nil
}
