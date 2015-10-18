package polochon

import "errors"

// Registerd modules
var registeredModules *RegisteredModules

func init() {
	registeredModules = &RegisteredModules{
		Detailers:   make(map[string]func(params map[string]interface{}) (Detailer, error)),
		Torrenters:  make(map[string]func(params map[string]interface{}) (Torrenter, error)),
		Guessers:    make(map[string]func(params map[string]interface{}) (Guesser, error)),
		FsNotifiers: make(map[string]func(params map[string]interface{}) (FsNotifier, error)),
		Notifiers:   make(map[string]func(params map[string]interface{}) (Notifier, error)),
		Subtitlers:  make(map[string]func(params map[string]interface{}) (Subtitler, error)),
		Wishlisters: make(map[string]func(params map[string]interface{}) (Wishlister, error)),
		Downloaders: make(map[string]func(params map[string]interface{}) (Downloader, error)),
		Calendars:   make(map[string]func(params map[string]interface{}) (Calendar, error)),
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
	TypeCalendar              = "calendar"
)

// RegisteredModules holds the modules registered during the init process
type RegisteredModules struct {
	Detailers   map[string]func(params map[string]interface{}) (Detailer, error)
	Torrenters  map[string]func(params map[string]interface{}) (Torrenter, error)
	Guessers    map[string]func(params map[string]interface{}) (Guesser, error)
	FsNotifiers map[string]func(params map[string]interface{}) (FsNotifier, error)
	Notifiers   map[string]func(params map[string]interface{}) (Notifier, error)
	Subtitlers  map[string]func(params map[string]interface{}) (Subtitler, error)
	Wishlisters map[string]func(params map[string]interface{}) (Wishlister, error)
	Downloaders map[string]func(params map[string]interface{}) (Downloader, error)
	Calendars   map[string]func(params map[string]interface{}) (Calendar, error)
}

// ConfigureDetailer configures a detailer
func ConfigureDetailer(name string, params map[string]interface{}) (Detailer, error) {
	f, ok := registeredModules.Detailers[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureSubtitler configures a subtitiler
func ConfigureSubtitler(name string, params map[string]interface{}) (Subtitler, error) {
	f, ok := registeredModules.Subtitlers[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureWishlister configures a wishlister
func ConfigureWishlister(name string, params map[string]interface{}) (Wishlister, error) {
	f, ok := registeredModules.Wishlisters[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureTorrenter configures a torrenter
func ConfigureTorrenter(name string, params map[string]interface{}) (Torrenter, error) {
	f, ok := registeredModules.Torrenters[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureGuesser configures a guesser
func ConfigureGuesser(name string, params map[string]interface{}) (Guesser, error) {
	f, ok := registeredModules.Guessers[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureFsNotifier configures a fs notifier
func ConfigureFsNotifier(name string, params map[string]interface{}) (FsNotifier, error) {
	f, ok := registeredModules.FsNotifiers[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureNotifier configures a notifier
func ConfigureNotifier(name string, params map[string]interface{}) (Notifier, error) {
	f, ok := registeredModules.Notifiers[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureDownloader configures a downloader
func ConfigureDownloader(name string, params map[string]interface{}) (Downloader, error) {
	f, ok := registeredModules.Downloaders[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureShowCalendarFetcher configures a show calendar fetcher
func ConfigureCalendar(name string, params map[string]interface{}) (Calendar, error) {
	f, ok := registeredModules.Calendars[name]
	if !ok {
		return nil, ErrModuleNotFound
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}
