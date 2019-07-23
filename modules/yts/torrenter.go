package yts

import (
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/yts"
	"github.com/sirupsen/logrus"
)

// GetTorrents implements the Torrenter interface
func (y *Yts) GetTorrents(i interface{}, log *logrus.Entry) error {
	m, err := getMovieArgument(i)
	if err != nil {
		return err
	}

	// matches returns a list of matching movie
	matches, err := searchByImdbID(m.ImdbID)
	if err != nil {
		return err
	}

	if len(matches) == 0 {
		return polochon.ErrMovieTorrentNotFound
	}

	// since we searched by id, there should be only one movie in the list
	ytsMovie := matches[0]

	// Check the torrent
	if len(ytsMovie.Torrents) == 0 {
		return polochon.ErrMovieTorrentNotFound
	}

	m.Torrents = polochonTorrents(&ytsMovie)

	return nil
}

// SearchTorrents implements the Torrenter interface
func (y *Yts) SearchTorrents(s string) ([]*polochon.Torrent, error) {
	movies, err := yts.Search(s)
	if err != nil {
		return nil, err
	}

	torrents := []*polochon.Torrent{}
	for _, m := range movies {
		pt := polochonTorrents(&m)
		if pt == nil {
			continue
		}

		for i := range pt {
			torrents = append(torrents, &pt[i])
		}
	}

	return torrents, nil
}
