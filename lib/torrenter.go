package polochon

import (
	"context"
	"errors"
)

// Torrenter error
var (
	ErrTorrentNotFound = errors.New("torrenter: failed to find torrent")
)

// Torrenter is an interface which allows to get torrent for a movie or a show
type Torrenter interface {
	Module
	GetTorrents(ctx context.Context, v any) error
	SearchTorrents(string) ([]*Torrent, error)
}

// Torrentable represents a resource which can be torrented
type Torrentable interface {
	GetTorrenters() []Torrenter
}

// GetTorrents helps getting the torrent files for a movie
func GetTorrents(ctx context.Context, v Torrentable) error {
	for _, t := range v.GetTorrenters() {
		if err := t.GetTorrents(ctx, v); err == nil {
			return nil
		}
	}

	return ErrTorrentNotFound
}
