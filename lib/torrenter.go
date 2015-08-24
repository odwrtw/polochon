package polochon

import (
	"errors"
	"log"

	"github.com/Sirupsen/logrus"
)

// Torrenter error
var (
	ErrTorrentNotFound            = errors.New("torrenter: failed to find torrent")
	ErrShowEpisodeTorrentNotFound = errors.New("torrenter: show episode torrent not found")
	ErrMovieTorrentNotFound       = errors.New("torrenter: movie torrent not found")
)

// Torrenter is an interface which allows to get torrent for a movie or a show
type Torrenter interface {
	Module
	GetTorrents(i interface{}) error
}

// RegisterTorrenter helps register a new torrenter
func RegisterTorrenter(name string, f func(params map[string]interface{}, log *logrus.Entry) (Torrenter, error)) {
	if _, ok := registeredModules.Torrenters[name]; ok {
		log.Panicf("modules: %q of type %q is already registered", name, TypeTorrenter)
	}

	// Register the module
	registeredModules.Torrenters[name] = f
}
