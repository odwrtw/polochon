// +build linux

package inotify

import (
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
	"golang.org/x/exp/inotify"
)

// Time to wait before sending an event
const DELAY time.Duration = 100 * time.Millisecond

// Module constants
const (
	moduleName = "inotify"
)

// Register fsnotify as a FsNotifier
func init() {
	polochon.RegisterFsNotifier(moduleName, NewInotify)
}

// NewInotify returns a new Inotify
func NewInotify(p []byte) (polochon.FsNotifier, error) {
	// Create a new inotify watcher
	watcher, err := inotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Inotifier{
		Events: []uint32{
			// File created
			inotify.IN_CREATE,
			// Directory created
			inotify.IN_CREATE + inotify.IN_ISDIR,
			// File moved
			inotify.IN_MOVE,
			inotify.IN_MOVED_TO,
			inotify.IN_MOVE_SELF,
			// Directory moved
			inotify.IN_MOVE + inotify.IN_ISDIR,
			inotify.IN_MOVED_TO + inotify.IN_ISDIR,
			inotify.IN_MOVE_SELF + inotify.IN_ISDIR,
		},
		watcher: watcher,
	}, nil
}

// Inotifier is a fsNotifier watching a directory
type Inotifier struct {
	// inotify event list
	Events  []uint32
	watcher *inotify.Watcher
}

// Name implements the Module interface
func (i *Inotifier) Name() string {
	return moduleName
}

// Watch start watching all the paths
func (i *Inotifier) Watch(pathToWatch string, ctx polochon.FsNotifierCtx, log *logrus.Entry) error {
	// Ensure that the watch path exists
	if _, err := os.Stat(pathToWatch); os.IsNotExist(err) {
		return err
	}

	// Run the event handler
	go i.eventHandler(ctx, log)

	// Check the path with inotify
	return i.watcher.Watch(pathToWatch)
}

func (i *Inotifier) eventHandler(ctx polochon.FsNotifierCtx, log *logrus.Entry) {
	// Notify the waitgroup
	ctx.Wg.Add(1)
	defer ctx.Wg.Done()

	// Close the watcher when done
	defer i.watcher.Close()

	for {
		select {
		case e := <-i.watcher.Event:
			// Check the current event in the event list
			for _, eventMask := range i.Events {
				if eventMask == e.Mask {
					log.Debug("File detected:", e.Name)
					// Wait for the delay time before sending an event.
					// Transmission creates the folder and move the files afterwards.
					// We need to wait for the file to be moved in before sending the
					// event. Delay is the estimated time to wait.
					go func() {
						time.Sleep(DELAY)
						ctx.Event <- e.Name
					}()
					break
				}
			}
		case err := <-i.watcher.Error:
			ctx.Errc <- err
		case <-ctx.Done:
			log.Info("inotify is done watching")
			return
		}
	}
}
