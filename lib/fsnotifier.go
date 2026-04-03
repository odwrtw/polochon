package polochon

import "sync"

// FsNotifierCtx is the context of the notifier
type FsNotifierCtx struct {
	Event chan string
	Done  <-chan struct{}
	Wg    *sync.WaitGroup
}

// FsNotifier is an interface to notify on filesystem change
type FsNotifier interface {
	Module
	Watch(watchPath string, ctx FsNotifierCtx) error
}
