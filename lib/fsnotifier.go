package polochon

import (
	"log"
	"sync"

	"github.com/Sirupsen/logrus"
)

// FsNotifierCtx is the context of the notifier
type FsNotifierCtx struct {
	Event chan string
	Done  chan struct{}
	Errc  chan error
	Wg    *sync.WaitGroup
}

// FsNotifier is an interface to notify on filesytem change
type FsNotifier interface {
	Watch(watchPath string, ctx FsNotifierCtx) error
}

// RegisterFsNotifier helps register a new FsNotifier
func RegisterFsNotifier(name string, f func(params map[string]string, log *logrus.Entry) (FsNotifier, error)) {
	if _, ok := registeredModules.FsNotifiers[name]; ok {
		log.Panicf("modules: %q of type %q is already registered", name, TypeFsNotifier)
	}

	// Register the module
	registeredModules.FsNotifiers[name] = f
}
