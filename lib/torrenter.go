package polochon

import (
	"errors"
	"fmt"

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
	GetTorrents(interface{}, *logrus.Entry) error
}

// RegisterTorrenter helps register a new torrenter
func RegisterTorrenter(name string, f func(params []byte) (Torrenter, error)) {
	if _, ok := registeredModules.Torrenters[name]; ok {
		panic(fmt.Sprintf("modules: %q of type %q is already registered", name, TypeDetailer))
	}

	// Register the module
	registeredModules.Torrenters[name] = f
}
