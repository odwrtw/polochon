package fsnotify

import (
	"os"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
	"gopkg.in/fsnotify.v1"
)

// Make sure that the module is a subtitler
var _ polochon.FsNotifier = (*FsNotify)(nil)

// Register fsnotify as a FsNotifier
func init() {
	polochon.RegisterModule(&FsNotify{})
}

const (
	// Module name
	moduleName = "fsnotify"

	// DELAY represents the time to wait before sending an event
	DELAY time.Duration = 100 * time.Millisecond
)

// FsNotify is a fsNotifier watching a directory
type FsNotify struct {
	watcher *fsnotify.Watcher
}

// Name implements the Module interface
func (fs *FsNotify) Name() string {
	return moduleName
}

// Status implements the Module interface
func (fs *FsNotify) Status() (polochon.ModuleStatus, error) {
	return polochon.StatusNotImplemented, nil
}

// Init implements the Module interface
func (fs *FsNotify) Init([]byte) error {
	return nil
}

// Watch implements the modules fsNotifier interface
func (fs *FsNotify) Watch(watchPath string, ctx polochon.FsNotifierCtx, log *logrus.Entry) error {
	// Create a new watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	fs.watcher = watcher

	// Ensure that the watch path exists
	if _, err := os.Stat(watchPath); os.IsNotExist(err) {
		return err
	}

	log = log.WithField("module", moduleName)

	// Run the event handler
	go fs.eventHandler(ctx, log)

	// Watch the path
	return fs.watcher.Add(watchPath)
}

func (fs *FsNotify) eventHandler(ctx polochon.FsNotifierCtx, log *logrus.Entry) {
	// Notify the waitgroup
	ctx.Wg.Add(1)
	defer ctx.Wg.Done()

	// Close the watcher when done
	defer fs.watcher.Close()

	for {
		select {
		case <-ctx.Done:
			log.Debug("fsnotify is done watching")
			return
		case ev := <-fs.watcher.Events:
			if ev.Op != fsnotify.Create && ev.Op != fsnotify.Chmod {
				continue
			}

			// Wait for the delay time before sending an event.
			// Transmission creates the folder and move the files afterwards.
			// We need to wait for the file to be moved in before sending the
			// event. Delay is the estimated time to wait.
			go func() {
				time.Sleep(DELAY)
				ctx.Event <- ev.Name
			}()
		case err := <-fs.watcher.Errors:
			log.Error(err)
		}
	}
}
