package mock

import (
	"errors"

	polochon "github.com/odwrtw/polochon/lib"
)

// Make sure that the module is a subtitler and a detailer
var (
	_ polochon.Subtitler = (*Mock)(nil)
	_ polochon.Detailer  = (*Mock)(nil)
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
