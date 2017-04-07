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
	Remove(Downloadable) error
	List() ([]Downloadable, error)
}

// Downloadable is an interface for anything to be downlaoded
type Downloadable interface {
	Infos() *DownloadableInfos
}

// DownloadableInfos represent infos about a Downloadable object
type DownloadableInfos struct {
	Ratio           float32
	IsFinished      bool
	FilePaths       []string
	Name            string
	AdditionalInfos map[string]interface{}
}

// RegisterDownloader helps register a new Downloader
func RegisterDownloader(name string, f func(params []byte) (Downloader, error)) {
	if _, ok := registeredModules.Downloaders[name]; ok {
		panic(fmt.Sprintf("modules: %q of type %q is already registered", name, TypeDownloader))
	}

	// Register the module
	registeredModules.Downloaders[name] = f
}
