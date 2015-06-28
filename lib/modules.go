package polochon

import (
	"errors"

	"github.com/Sirupsen/logrus"
)

// Registerd modules
var registeredModules *RegisteredModules
var configuredModules *Modules = NewModules()

func init() {
	registeredModules = &RegisteredModules{
		Detailers:   make(map[string]func(params map[string]string, log *logrus.Entry) (Detailer, error)),
		Torrenters:  make(map[string]func(params map[string]string, log *logrus.Entry) (Torrenter, error)),
		Guessers:    make(map[string]func(params map[string]string, log *logrus.Entry) (Guesser, error)),
		FsNotifiers: make(map[string]func(params map[string]string, log *logrus.Entry) (FsNotifier, error)),
		Notifiers:   make(map[string]func(params map[string]string, log *logrus.Entry) (Notifier, error)),
		Subtitilers: make(map[string]func(params map[string]string, log *logrus.Entry) (Subtitiler, error)),
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
	Detailers   map[string]func(params map[string]string, log *logrus.Entry) (Detailer, error)
	Torrenters  map[string]func(params map[string]string, log *logrus.Entry) (Torrenter, error)
	Guessers    map[string]func(params map[string]string, log *logrus.Entry) (Guesser, error)
	FsNotifiers map[string]func(params map[string]string, log *logrus.Entry) (FsNotifier, error)
	Notifiers   map[string]func(params map[string]string, log *logrus.Entry) (Notifier, error)
	Subtitilers map[string]func(params map[string]string, log *logrus.Entry) (Subtitiler, error)
}

// Modules holds the configured modules
type Modules struct {
	Detailers   map[string]Detailer
	Torrenters  map[string]Torrenter
	Guessers    map[string]Guesser
	FsNotifiers map[string]FsNotifier
	Notifiers   map[string]Notifier
	Subtitilers map[string]Subtitiler
}

// NewModules returns a new set of modules
func NewModules() *Modules {
	return &Modules{
		Detailers:   make(map[string]Detailer),
		Torrenters:  make(map[string]Torrenter),
		Guessers:    make(map[string]Guesser),
		FsNotifiers: make(map[string]FsNotifier),
		Notifiers:   make(map[string]Notifier),
		Subtitilers: make(map[string]Subtitiler),
	}
}

// ConfigureDetailer configures a detailer
func (m *Modules) ConfigureDetailer(name string, params map[string]string, log *logrus.Entry) error {
	f, ok := registeredModules.Detailers[name]
	if !ok {
		return ErrModuleNotFound
	}

	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeDetailer})

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return err
	}
	m.Detailers[name] = module

	return nil
}

// ConfigureSubtitler configures a subtitiler
func (m *Modules) ConfigureSubtitler(name string, params map[string]string, log *logrus.Entry) error {
	f, ok := registeredModules.Subtitilers[name]
	if !ok {
		return ErrModuleNotFound
	}

	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeDetailer})

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return err
	}
	m.Subtitilers[name] = module

	return nil
}

// ConfigureTorrenter configures a torrenter
func (m *Modules) ConfigureTorrenter(name string, params map[string]string, log *logrus.Entry) error {
	f, ok := registeredModules.Torrenters[name]
	if !ok {
		return ErrModuleNotFound
	}

	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeDetailer})

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return err
	}
	m.Torrenters[name] = module

	return nil
}

// ConfigureGuesser configures a guesser
func (m *Modules) ConfigureGuesser(name string, params map[string]string, log *logrus.Entry) error {
	f, ok := registeredModules.Guessers[name]
	if !ok {
		return ErrModuleNotFound
	}

	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeDetailer})

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return err
	}
	m.Guessers[name] = module

	return nil
}

// ConfigureFsNotifier configures a fs notifier
func (m *Modules) ConfigureFsNotifier(name string, params map[string]string, log *logrus.Entry) error {
	f, ok := registeredModules.FsNotifiers[name]
	if !ok {
		return ErrModuleNotFound
	}

	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeDetailer})

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return err
	}
	m.FsNotifiers[name] = module

	return nil
}

// ConfigureNotifier configures a notifier
func (m *Modules) ConfigureNotifier(name string, params map[string]string, log *logrus.Entry) error {
	f, ok := registeredModules.Notifiers[name]
	if !ok {
		return ErrModuleNotFound
	}

	// Setup the logs
	logger := log.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeDetailer})

	// Configure the module
	module, err := f(params, logger)
	if err != nil {
		return err
	}
	m.Notifiers[name] = module

	return nil
}
