package mock

import (
	"errors"
	"fmt"
	"math/rand"

	polochon "github.com/odwrtw/polochon/lib"
)

// Make sure that the module is a subtitler and a detailer
var (
	_ polochon.Torrenter  = (*Mock)(nil)
	_ polochon.Detailer   = (*Mock)(nil)
	_ polochon.Guesser    = (*Mock)(nil)
	_ polochon.FsNotifier = (*Mock)(nil)
	_ polochon.Notifier   = (*Mock)(nil)
	_ polochon.Subtitler  = (*Mock)(nil)
	_ polochon.Wishlister = (*Mock)(nil)
	_ polochon.Downloader = (*Mock)(nil)
	_ polochon.Calendar   = (*Mock)(nil)
	_ polochon.Explorer   = (*Mock)(nil)
	_ polochon.Searcher   = (*Mock)(nil)
)

func init() {
	polochon.RegisterModule(&Mock{})
}

// Module constants
const (
	moduleName = "mock"
)

// Custom errors
var (
	ErrInvalidArgument = errors.New("mock: invalid argument type")
)

// Mock is a mock module for test purposes
type Mock struct{}

// Init implements the Module interface
func (mock *Mock) Init(p []byte) error {
	return nil
}

// Name implements the Module interface
func (mock *Mock) Name() string {
	return moduleName
}

// Status implements the Module interface
func (mock *Mock) Status() (polochon.ModuleStatus, error) {
	return polochon.StatusOK, nil
}

func randomImdbID() string {
	return fmt.Sprintf("tt%07d", rand.Intn(999999))
}
