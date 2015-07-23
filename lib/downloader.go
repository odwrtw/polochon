package polochon

import (
	"log"

	"github.com/Sirupsen/logrus"
)

// Downloader represent a interface for any downloader
type Downloader interface {
	Module
	Download(URL string) error
}

// Downloadable is an interface for anything to be downlaoded
type Downloadable interface {
	Status() error
	Remove() error
	Download() error
}

// RegisterDownloader helps register a new Downloader
func RegisterDownloader(name string, f func(params map[string]interface{}, log *logrus.Entry) (Downloader, error)) {
	if _, ok := registeredModules.Downloaders[name]; ok {
		log.Panicf("modules: %q of type %q is already registered", name, TypeDownloader)
	}

	// Register the module
	registeredModules.Downloaders[name] = f
}
