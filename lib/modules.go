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
		Subtitilers: make(map[string]func(params map[string]interface{}, log *logrus.Entry) (Subtitiler, error)),
	}
}

// Modules error
var (
	ErrNoModuleFound  = errors.New("modules: no module found")
	ErrModuleNotFound = errors.New("modules: module not found")
)

// ModuleType holds modules type
type ModuleType string

// Available modules types
const (
	TypeTorrenter  ModuleType = "torrenter"
	TypeDetailer              = "detailer"
	TypeGuesser               = "guesser"
	TypeFsNotifier            = "fsnotifier"
	TypeNotifier              = "notifier"
	TypeSubtitiler            = "subtitiler"
)

// RegisteredModules holds the modules registered during the init process
type RegisteredModules struct {
	Detailers   map[string]func(params map[string]interface{}, log *logrus.Entry) (Detailer, error)
	Torrenters  map[string]func(params map[string]interface{}, log *logrus.Entry) (Torrenter, error)
	Guessers    map[string]func(params map[string]interface{}, log *logrus.Entry) (Guesser, error)
	FsNotifiers map[string]func(params map[string]interface{}, log *logrus.Entry) (FsNotifier, error)
	Notifiers   map[string]func(params map[string]interface{}, log *logrus.Entry) (Notifier, error)
	Subtitilers map[string]func(params map[string]interface{}, log *logrus.Entry) (Subtitiler, error)
}

// ConfigureDetailer configures a detailer
func ConfigureDetailer(name string, params map[string]interface{}, log *logrus.Entry) (Detailer, error) {
	f, ok := registeredModules.Detailers[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeDetailer})

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureSubtitler configures a subtitiler
func ConfigureSubtitler(name string, params map[string]interface{}, log *logrus.Entry) (Subtitiler, error) {
	f, ok := registeredModules.Subtitilers[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeDetailer})

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureTorrenter configures a torrenter
func ConfigureTorrenter(name string, params map[string]interface{}, log *logrus.Entry) (Torrenter, error) {
	f, ok := registeredModules.Torrenters[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeDetailer})

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureGuesser configures a guesser
func ConfigureGuesser(name string, params map[string]interface{}, log *logrus.Entry) (Guesser, error) {
	f, ok := registeredModules.Guessers[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeDetailer})

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureFsNotifier configures a fs notifier
func ConfigureFsNotifier(name string, params map[string]interface{}, log *logrus.Entry) (FsNotifier, error) {
	f, ok := registeredModules.FsNotifiers[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeDetailer})

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureNotifier configures a notifier
func ConfigureNotifier(name string, params map[string]interface{}, log *logrus.Entry) (Notifier, error) {
	f, ok := registeredModules.Notifiers[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeDetailer})

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return nil, err
	}

	return module, nil
}
