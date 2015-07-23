package polochon

import (
	"errors"

	"github.com/Sirupsen/logrus"
)

// Registerd modules
var registeredModules *RegisteredModules

func init() {
	registeredModules = &RegisteredModules{
		Detailers:   make(map[string]func(params map[string]interface{}, log *logrus.Entry) (Detailer, error)),
		Torrenters:  make(map[string]func(params map[string]interface{}, log *logrus.Entry) (Torrenter, error)),
		Guessers:    make(map[string]func(params map[string]interface{}, log *logrus.Entry) (Guesser, error)),
		FsNotifiers: make(map[string]func(params map[string]interface{}, log *logrus.Entry) (FsNotifier, error)),
		Notifiers:   make(map[string]func(params map[string]interface{}, log *logrus.Entry) (Notifier, error)),
		Subtitlers:  make(map[string]func(params map[string]interface{}, log *logrus.Entry) (Subtitler, error)),
		Wishlisters: make(map[string]func(params map[string]interface{}, log *logrus.Entry) (Wishlister, error)),
		Downloaders: make(map[string]func(params map[string]interface{}, log *logrus.Entry) (Downloader, error)),
	}
}

// Modules error
var (
	ErrNoModuleFound  = errors.New("modules: no module found")
	ErrModuleNotFound = errors.New("modules: module not found")
)

// ModuleType holds modules type
type ModuleType string

// Module type, all modules must implement it
type Module interface {
	Name() string
}

// Available modules types
const (
	TypeTorrenter  ModuleType = "torrenter"
	TypeDetailer              = "detailer"
	TypeGuesser               = "guesser"
	TypeFsNotifier            = "fsnotifier"
	TypeNotifier              = "notifier"
	TypeSubtitler             = "subtitler"
	TypeWishlister            = "wishlister"
	TypeDownloader            = "downloader"
)

// RegisteredModules holds the modules registered during the init process
type RegisteredModules struct {
	Detailers   map[string]func(params map[string]interface{}, log *logrus.Entry) (Detailer, error)
	Torrenters  map[string]func(params map[string]interface{}, log *logrus.Entry) (Torrenter, error)
	Guessers    map[string]func(params map[string]interface{}, log *logrus.Entry) (Guesser, error)
	FsNotifiers map[string]func(params map[string]interface{}, log *logrus.Entry) (FsNotifier, error)
	Notifiers   map[string]func(params map[string]interface{}, log *logrus.Entry) (Notifier, error)
	Subtitlers  map[string]func(params map[string]interface{}, log *logrus.Entry) (Subtitler, error)
	Wishlisters map[string]func(params map[string]interface{}, log *logrus.Entry) (Wishlister, error)
	Downloaders map[string]func(params map[string]interface{}, log *logrus.Entry) (Downloader, error)
}

// ConfigureDetailer configures a detailer
func ConfigureDetailer(name string, params map[string]interface{}, log *logrus.Entry) (Detailer, error) {
	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeDetailer})

	f, ok := registeredModules.Detailers[name]
	if !ok {
		logger.Infof("No such module %q", name)
		return nil, ErrModuleNotFound
	}

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureSubtitler configures a subtitiler
func ConfigureSubtitler(name string, params map[string]interface{}, log *logrus.Entry) (Subtitler, error) {
	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeSubtitler})

	f, ok := registeredModules.Subtitlers[name]
	if !ok {
		logger.Infof("No such module %q", name)
		return nil, ErrModuleNotFound
	}

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureWishlister configures a wishlister
func ConfigureWishlister(name string, params map[string]interface{}, log *logrus.Entry) (Wishlister, error) {
	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeWishlister})

	f, ok := registeredModules.Wishlisters[name]
	if !ok {
		logger.Infof("No such module %q", name)
		return nil, ErrModuleNotFound
	}

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureTorrenter configures a torrenter
func ConfigureTorrenter(name string, params map[string]interface{}, log *logrus.Entry) (Torrenter, error) {
	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeTorrenter})

	f, ok := registeredModules.Torrenters[name]
	if !ok {
		logger.Infof("No such module %q", name)
		return nil, ErrModuleNotFound
	}

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureGuesser configures a guesser
func ConfigureGuesser(name string, params map[string]interface{}, log *logrus.Entry) (Guesser, error) {
	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeGuesser})

	f, ok := registeredModules.Guessers[name]
	if !ok {
		logger.Infof("No such module %q", name)
		return nil, ErrModuleNotFound
	}

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureFsNotifier configures a fs notifier
func ConfigureFsNotifier(name string, params map[string]interface{}, log *logrus.Entry) (FsNotifier, error) {
	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeFsNotifier})

	f, ok := registeredModules.FsNotifiers[name]
	if !ok {
		logger.Infof("No such module %q", name)
		return nil, ErrModuleNotFound
	}

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureNotifier configures a notifier
func ConfigureNotifier(name string, params map[string]interface{}, log *logrus.Entry) (Notifier, error) {
	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeNotifier})

	f, ok := registeredModules.Notifiers[name]
	if !ok {
		logger.Infof("No such module %q", name)
		return nil, ErrModuleNotFound
	}

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureDownloader configures a downloader
func ConfigureDownloader(name string, params map[string]interface{}, log *logrus.Entry) (Downloader, error) {
	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeDownloader})

	f, ok := registeredModules.Downloaders[name]
	if !ok {
		logger.Infof("No such module %q", name)
		return nil, ErrModuleNotFound
	}

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return nil, err
	}

	return module, nil
}
