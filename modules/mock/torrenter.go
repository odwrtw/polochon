package mock

import (
	"fmt"
	"math/rand"
	"strconv"

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
	e.Torrents = []*polochon.Torrent{}
	for _, q := range []polochon.Quality{polochon.Quality480p, polochon.Quality720p} {
		e.Torrents = append(e.Torrents, &polochon.Torrent{
			ImdbID:  "tt" + strconv.Itoa(rand.Intn(100)),
			Type:    "episode",
			Season:  rand.Intn(10),
			Episode: rand.Intn(10),
			Quality: q,
			Result: &polochon.TorrentResult{
				URL:      fmt.Sprintf("https://mock.com/t%s.torrent", q),
				Source:   moduleName,
				Seeders:  rand.Intn(100),
				Leechers: rand.Intn(500),
			},
		})
	}
}

func (mock *Mock) getMovieTorrents(m *polochon.Movie) {
	m.Torrents = []*polochon.Torrent{}
	for _, q := range []polochon.Quality{polochon.Quality720p, polochon.Quality1080p, polochon.Quality3D} {
		m.Torrents = append(m.Torrents, &polochon.Torrent{
			ImdbID:  "tt" + strconv.Itoa(rand.Intn(100)),
			Type:    "movie",
			Quality: q,
			Result: &polochon.TorrentResult{
				URL:      fmt.Sprintf("https://mock.com/t%s.torrent", q),
				Source:   moduleName,
				Seeders:  rand.Intn(100),
				Leechers: rand.Intn(500),
			},
		})
	}
}

// SearchTorrents implements the torrenter interface
func (mock *Mock) SearchTorrents(string) ([]*polochon.Torrent, error) {
	torrents := []*polochon.Torrent{}
	for i, q := range []polochon.Quality{polochon.Quality720p, polochon.Quality1080p, polochon.Quality3D} {
		torrents = append(torrents, &polochon.Torrent{
			Quality: q,
			Result: &polochon.TorrentResult{
				URL:      fmt.Sprintf("https://mock.com/t%s.torrent", q),
				Source:   moduleName,
				Seeders:  rand.Intn(100),
				Leechers: rand.Intn(500),
				Size:     1000000000 * (i + 1),
			},
		})
	}
	return torrents, nil
}
