package polochon

import (
	"errors"

	"github.com/sirupsen/logrus"
)

// Torrenter error
var (
	ErrTorrentNotFound = errors.New("torrenter: failed to find torrent")
)

// Torrenter is an interface which allows to get torrent for a movie or a show
type Torrenter interface {
	Module
	GetTorrents(interface{}, *logrus.Entry) error
	SearchTorrents(string) ([]*Torrent, error)
}

// Torrentable represents a resource which can be torrented
type Torrentable interface {
	GetTorrenters() []Torrenter
}

// GetTorrents helps getting the torrent files for a movie
func GetTorrents(v Torrentable, log *logrus.Entry) error {
	for _, t := range v.GetTorrenters() {
		torrenterLog := log.WithField("torrenter", t.Name())
		err := t.GetTorrents(v, torrenterLog)
		if err == nil {
			// Torrents found
			return nil
		}
	}

	return ErrTorrentNotFound
}
