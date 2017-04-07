package polochon

import (
	"fmt"
	"sync"

	"github.com/Sirupsen/logrus"
)

// FsNotifierCtx is the context of the notifier
type FsNotifierCtx struct {
	Event chan string
	Done  <-chan struct{}
	Wg    *sync.WaitGroup
}

// FsNotifier is an interface to notify on filesystem change
type FsNotifier interface {
	Module
	Watch(watchPath string, ctx FsNotifierCtx, log *logrus.Entry) error
}

// RegisterFsNotifier helps register a new FsNotifier
func RegisterFsNotifier(name string, f func(params []byte) (FsNotifier, error)) {
	if _, ok := registeredModules.FsNotifiers[name]; ok {
		panic(fmt.Sprintf("modules: %q of type %q is already registered", name, TypeNotifier))
	}

	// Register the module
	registeredModules.FsNotifiers[name] = f
}
