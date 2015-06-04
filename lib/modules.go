package polochon

import (
	"errors"

	"github.com/Sirupsen/logrus"
)

// Registerd modules
var registeredModules *RegisteredModules

func init() {
	registeredModules = &RegisteredModules{
		Detailers:   make(map[string]func(params map[string]string, log *logrus.Entry) (Detailer, error)),
		Torrenters:  make(map[string]func(params map[string]string, log *logrus.Entry) (Torrenter, error)),
		Guessers:    make(map[string]func(params map[string]string, log *logrus.Entry) (Guesser, error)),
		FsNotifiers: make(map[string]func(params map[string]string, log *logrus.Entry) (FsNotifier, error)),
		Notifiers:   make(map[string]func(params map[string]string, log *logrus.Entry) (Notifier, error)),
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
)

// RegisteredModules holds the modules registered during the init process
type RegisteredModules struct {
	Detailers   map[string]func(params map[string]string, log *logrus.Entry) (Detailer, error)
	Torrenters  map[string]func(params map[string]string, log *logrus.Entry) (Torrenter, error)
	Guessers    map[string]func(params map[string]string, log *logrus.Entry) (Guesser, error)
	FsNotifiers map[string]func(params map[string]string, log *logrus.Entry) (FsNotifier, error)
	Notifiers   map[string]func(params map[string]string, log *logrus.Entry) (Notifier, error)
}

// Modules holds the configured modules
type Modules struct {
	Logger      *logrus.Entry
	Detailers   map[string]Detailer
	Torrenters  map[string]Torrenter
	Guessers    map[string]Guesser
	FsNotifiers map[string]FsNotifier
	Notifiers   map[string]Notifier
}

// NewModules returns a new set of modules
func NewModules(logger *logrus.Entry) *Modules {
	return &Modules{
		Logger:      logger,
		Detailers:   make(map[string]Detailer),
		Torrenters:  make(map[string]Torrenter),
		Guessers:    make(map[string]Guesser),
		FsNotifiers: make(map[string]FsNotifier),
		Notifiers:   make(map[string]Notifier),
	}
}

// ConfigureDetailer configures a detailer
func (m *Modules) ConfigureDetailer(name string, params map[string]string) error {
	f, ok := registeredModules.Detailers[name]
	if !ok {
		return ErrModuleNotFound
	}

	// Setup the logs
	log := m.Logger.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeDetailer})

	// Configure the module
	module, err := f(params, log)
	if err != nil {
		return err
	}
	m.Detailers[name] = module

	return nil
}

// ConfigureTorrenter configures a torrenter
func (m *Modules) ConfigureTorrenter(name string, params map[string]string) error {
	f, ok := registeredModules.Torrenters[name]
	if !ok {
		return ErrModuleNotFound
	}

	// Setup the logs
	log := m.Logger.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeTorrenter})

	// Configure the module
	module, err := f(params, log)
	if err != nil {
		return err
	}
	m.Torrenters[name] = module

	return nil
}

// ConfigureGuesser configures a guesser
func (m *Modules) ConfigureGuesser(name string, params map[string]string) error {
	f, ok := registeredModules.Guessers[name]
	if !ok {
		return ErrModuleNotFound
	}

	// Setup the logs
	log := m.Logger.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeGuesser})

	// Configure the module
	module, err := f(params, log)
	if err != nil {
		return err
	}
	m.Guessers[name] = module

	return nil
}

// ConfigureFsNotifier configures a fs notifier
func (m *Modules) ConfigureFsNotifier(name string, params map[string]string) error {
	f, ok := registeredModules.FsNotifiers[name]
	if !ok {
		return ErrModuleNotFound
	}

	// Setup the logs
	log := m.Logger.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeFsNotifier})

	// Configure the module
	module, err := f(params, log)
	if err != nil {
		return err
	}
	m.FsNotifiers[name] = module

	return nil
}

// ConfigureNotifier configures a notifier
func (m *Modules) ConfigureNotifier(name string, params map[string]string) error {
	f, ok := registeredModules.Notifiers[name]
	if !ok {
		return ErrModuleNotFound
	}

	// Setup the logs
	log := m.Logger.WithFields(logrus.Fields{"moduleName": name, "moduleType": TypeNotifier})

	// Configure the module
	module, err := f(params, log)
	if err != nil {
		return err
	}
	m.Notifiers[name] = module

	return nil
}

// Detailer gets a configured Detailer
func (m *Modules) Detailer(name string) (Detailer, error) {
	module, ok := m.Detailers[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	return module, nil
}

// Torrenter gets a configured Torrenter
func (m *Modules) Torrenter(name string) (Torrenter, error) {
	module, ok := m.Torrenters[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	return module, nil
}

// Notifier gets a configured Notifier
func (m *Modules) Notifier(name string) (Notifier, error) {
	module, ok := m.Notifiers[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	return module, nil
}

// Guesser gets a configured Guesser
func (m *Modules) Guesser(name string) (Guesser, error) {
	module, ok := m.Guessers[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	return module, nil
}

// FsNotifier gets a configured FsNotifier
func (m *Modules) FsNotifier(name string) (FsNotifier, error) {
	module, ok := m.FsNotifiers[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	return module, nil
}
