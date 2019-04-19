package polochon

import (
	"errors"
	"fmt"
)

// Registered modules
var registeredModules *modules

func init() {
	registeredModules = &modules{
		Detailers:   make(map[string]func(params []byte) (Detailer, error)),
		Torrenters:  make(map[string]func(params []byte) (Torrenter, error)),
		Guessers:    make(map[string]func(params []byte) (Guesser, error)),
		FsNotifiers: make(map[string]func(params []byte) (FsNotifier, error)),
		Notifiers:   make(map[string]func(params []byte) (Notifier, error)),
		Subtitlers:  make(map[string]func(params []byte) (Subtitler, error)),
		Wishlisters: make(map[string]func(params []byte) (Wishlister, error)),
		Downloaders: make(map[string]func(params []byte) (Downloader, error)),
		Calendars:   make(map[string]func(params []byte) (Calendar, error)),
		Searchers:   make(map[string]func(params []byte) (Searcher, error)),
		Explorers:   make(map[string]func(params []byte) (Explorer, error)),
	}
}

// Modules error
var (
	ErrNoModuleFound = errors.New("modules: no module found")
	ErrNotAvailable  = errors.New("modules: function not available")
)

// ModuleType holds modules type
type ModuleType string

// ModuleStatus holds modules status
type ModuleStatus string

// Module type, all modules must implement it
type Module interface {
	Name() string
	Status() (ModuleStatus, error)
}

// Available modules types
const (
	TypeTorrenter  ModuleType = "torrenter"
	TypeDetailer   ModuleType = "detailer"
	TypeGuesser    ModuleType = "guesser"
	TypeFsNotifier ModuleType = "fsnotifier"
	TypeNotifier   ModuleType = "notifier"
	TypeSubtitler  ModuleType = "subtitler"
	TypeWishlister ModuleType = "wishlister"
	TypeDownloader ModuleType = "downloader"
	TypeCalendar   ModuleType = "calendar"
	TypeExplorer   ModuleType = "explorer"
	TypeSearcher   ModuleType = "searcher"
)

// Available modules statuses
const (
	StatusOK             ModuleStatus = "ok"
	StatusNotImplemented ModuleStatus = "not_implemented"
	StatusFail           ModuleStatus = "fail"
)

// modules holds the modules registered during the init process
type modules struct {
	Detailers   map[string]func(params []byte) (Detailer, error)
	Torrenters  map[string]func(params []byte) (Torrenter, error)
	Guessers    map[string]func(params []byte) (Guesser, error)
	FsNotifiers map[string]func(params []byte) (FsNotifier, error)
	Notifiers   map[string]func(params []byte) (Notifier, error)
	Subtitlers  map[string]func(params []byte) (Subtitler, error)
	Wishlisters map[string]func(params []byte) (Wishlister, error)
	Downloaders map[string]func(params []byte) (Downloader, error)
	Calendars   map[string]func(params []byte) (Calendar, error)
	Searchers   map[string]func(params []byte) (Searcher, error)
	Explorers   map[string]func(params []byte) (Explorer, error)
}

// ConfigureDetailer configures a detailer
func ConfigureDetailer(name string, params []byte) (Detailer, error) {
	f, ok := registeredModules.Detailers[name]
	if !ok {
		return nil, fmt.Errorf("modules: detailer module '%s' not found", name)
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureSubtitler configures a subtitiler
func ConfigureSubtitler(name string, params []byte) (Subtitler, error) {
	f, ok := registeredModules.Subtitlers[name]
	if !ok {
		return nil, fmt.Errorf("modules: subtitler module '%s' not found", name)
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureWishlister configures a wishlister
func ConfigureWishlister(name string, params []byte) (Wishlister, error) {
	f, ok := registeredModules.Wishlisters[name]
	if !ok {
		return nil, fmt.Errorf("modules: wishilsit module '%s' not found", name)
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureTorrenter configures a torrenter
func ConfigureTorrenter(name string, params []byte) (Torrenter, error) {
	f, ok := registeredModules.Torrenters[name]
	if !ok {
		return nil, fmt.Errorf("modules: torrenter module '%s' not found", name)
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureGuesser configures a guesser
func ConfigureGuesser(name string, params []byte) (Guesser, error) {
	f, ok := registeredModules.Guessers[name]
	if !ok {
		return nil, fmt.Errorf("modules: guesser module '%s' not found", name)
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureFsNotifier configures a fs notifier
func ConfigureFsNotifier(name string, params []byte) (FsNotifier, error) {
	f, ok := registeredModules.FsNotifiers[name]
	if !ok {
		return nil, fmt.Errorf("modules: fsnotifier module '%s' not found", name)
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureNotifier configures a notifier
func ConfigureNotifier(name string, params []byte) (Notifier, error) {
	f, ok := registeredModules.Notifiers[name]
	if !ok {
		return nil, fmt.Errorf("modules: notifier module '%s' not found", name)
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureDownloader configures a downloader
func ConfigureDownloader(name string, params []byte) (Downloader, error) {
	f, ok := registeredModules.Downloaders[name]
	if !ok {
		return nil, fmt.Errorf("modules: downloader module '%s' not found", name)
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureCalendar configures a show calendar fetcher
func ConfigureCalendar(name string, params []byte) (Calendar, error) {
	f, ok := registeredModules.Calendars[name]
	if !ok {
		return nil, fmt.Errorf("modules: calendar module '%s' not found", name)
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureExplorer configures an explorer
func ConfigureExplorer(name string, params []byte) (Explorer, error) {
	f, ok := registeredModules.Explorers[name]
	if !ok {
		return nil, fmt.Errorf("modules: explorer module '%s' not found", name)
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}

// ConfigureSearcher configures a searcher
func ConfigureSearcher(name string, params []byte) (Searcher, error) {
	f, ok := registeredModules.Searchers[name]
	if !ok {
		return nil, fmt.Errorf("modules: searcher module '%s' not found", name)
	}

	// Configure the module
	module, err := f(params)
	if err != nil {
		return nil, err
	}

	return module, nil
}
