package polochon

import (
	"sync"

	"github.com/sirupsen/logrus"
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
