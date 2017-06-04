package polochon

import (
	"errors"
	"fmt"

	polochonError "github.com/odwrtw/errors"
	"github.com/sirupsen/logrus"
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

// Torrentable represents a ressource which can be torrented
type Torrentable interface {
	GetTorrenters() []Torrenter
}

// RegisterTorrenter helps register a new torrenter
func RegisterTorrenter(name string, f func(params []byte) (Torrenter, error)) {
	if _, ok := registeredModules.Torrenters[name]; ok {
		panic(fmt.Sprintf("modules: %q of type %q is already registered", name, TypeTorrenter))
	}

	// Register the module
	registeredModules.Torrenters[name] = f
}

// GetTorrents helps getting the torrent files for a movie
// If there is an error, it will be of type *errors.Collector
func GetTorrents(v Torrentable, log *logrus.Entry) error {
	c := polochonError.NewCollector()

	for _, t := range v.GetTorrenters() {
		torrenterLog := log.WithField("torrenter", t.Name())
		err := t.GetTorrents(v, torrenterLog)
		if err == nil {
			break
		}
		c.Push(polochonError.Wrap(err).Ctx("Torrenter", t.Name()))
	}

	if c.HasErrors() {
		return c
	}

	return nil
}
