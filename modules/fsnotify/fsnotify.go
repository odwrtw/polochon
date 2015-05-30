package fsnotify

import (
	"os"

	"github.com/Sirupsen/logrus"
	"gitlab.quimbo.fr/odwrtw/polochon/lib"
	"gopkg.in/fsnotify.v1"
)

// Register fsnotify as a FsNotifier
func init() {
	polochon.RegisterFsNotifier("fsnotify", NewFsNotify)
}

// FsNotify is a fsNotifier watching a directory
type FsNotify struct {
	log     *logrus.Entry
	watcher *fsnotify.Watcher
}

// NewFsNotify returns a new FsNotify
func NewFsNotify(params map[string]string, log *logrus.Entry) (polochon.FsNotifier, error) {
	// Create a new watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &FsNotify{
		log:     log,
		watcher: watcher,
	}, nil
}

// Watch implements the modules fsNotifier interface
func (fs *FsNotify) Watch(watchPath string, ctx polochon.FsNotifierCtx) error {
	// Ensure that the watch path exists
	if _, err := os.Stat(watchPath); os.IsNotExist(err) {
		return err
	}

	// Run the event handler
	go fs.eventHandler(ctx)

	// Watch the path
	if err := fs.watcher.Add(watchPath); err != nil {
		return err
	}

	return nil
}

func (fs *FsNotify) eventHandler(ctx polochon.FsNotifierCtx) {
	// Notify the waitgroup
	ctx.Wg.Add(1)
	defer ctx.Wg.Done()

	// Close the watcher when done
	defer fs.watcher.Close()

	for {
		select {
		case <-ctx.Done:
			fs.log.Info("fsnotify is done watching")
			return
		case ev := <-fs.watcher.Events:
			if ev.Op != fsnotify.Create && ev.Op != fsnotify.Chmod {
				continue
			}
			ctx.Event <- ev.Name
		case err := <-fs.watcher.Errors:
			ctx.Errc <- err
		}
	}
}
