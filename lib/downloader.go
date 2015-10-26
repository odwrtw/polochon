package polochon

import (
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"
)

var (
	// ErrDuplicateTorrent returned when the torrent is already added
	ErrDuplicateTorrent = errors.New("Torrent already added")
)

// Downloader represent a interface for any downloader
type Downloader interface {
	Module
	Download(string, *logrus.Entry) error
}

// Downloadable is an interface for anything to be downlaoded
type Downloadable interface {
	Status(*logrus.Entry) error
	Remove(*logrus.Entry) error
	Download(*logrus.Entry) error
}

// RegisterDownloader helps register a new Downloader
func RegisterDownloader(name string, f func(params []byte) (Downloader, error)) {
	if _, ok := registeredModules.Downloaders[name]; ok {
		panic(fmt.Sprintf("modules: %q of type %q is already registered", name, TypeDetailer))
	}

	// Register the module
	registeredModules.Downloaders[name] = f
}
