package polochon

import (
	"errors"
	"fmt"
	"reflect"
)

// Registered modules
var registeredModules = map[string]Module{}

// Modules error
var (
	ErrNotAvailable = errors.New("modules: function not available")
)

// ModuleType holds modules type
type ModuleType string

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

// ModuleStatus holds modules status
type ModuleStatus string

// Module type, all modules must implement it
type Module interface {
	Init(params []byte) error
	Name() string
	Status() (ModuleStatus, error)
}

// Available modules statuses
const (
	StatusOK             ModuleStatus = "ok"
	StatusNotImplemented ModuleStatus = "not_implemented"
	StatusFail           ModuleStatus = "fail"
)

// RegisterModule is a function used by the module to register themselves
func RegisterModule(m Module) {
	if m.Name() == "" {
		panic("modules: missing module name")
	}

	if _, ok := registeredModules[m.Name()]; ok {
		panic(fmt.Sprintf("modules: %s is already registered", m.Name()))
	}

	registeredModules[m.Name()] = m
}

var moduleTypeToInterface = map[ModuleType]interface{}{
	TypeCalendar:   (*Calendar)(nil),
	TypeDetailer:   (*Detailer)(nil),
	TypeDownloader: (*Downloader)(nil),
	TypeExplorer:   (*Explorer)(nil),
	TypeFsNotifier: (*FsNotifier)(nil),
	TypeGuesser:    (*Guesser)(nil),
	TypeNotifier:   (*Notifier)(nil),
	TypeSearcher:   (*Searcher)(nil),
	TypeSubtitler:  (*Subtitler)(nil),
	TypeTorrenter:  (*Torrenter)(nil),
	TypeWishlister: (*Wishlister)(nil),
}

// GetModule returns a module ensuring it implements the type linked to the
// ModuleType
func GetModule(name string, t ModuleType) (Module, error) {
	m, ok := registeredModules[name]
	if !ok {
		return nil, fmt.Errorf("polochon: module %s not found", name)
	}

	interfaceType, ok := moduleTypeToInterface[t]
	if !ok {
		return nil, fmt.Errorf("polochon: module type %s not found", string(t))
	}

	moduleType := reflect.TypeOf(m)
	if !moduleType.Implements(reflect.TypeOf(interfaceType).Elem()) {
		return nil, fmt.Errorf("polochon: module %s does not implement the interface %s", name, string(t))
	}

	return m, nil
}

// ClearRegisteredModules clears the registered modules from polochon. This
// function exists for test purposes only. Do not use this unless you know what
// you're doing.
func ClearRegisteredModules() {
	registeredModules = map[string]Module{}
}
