package organizer

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/errors"
	"github.com/odwrtw/polochon/app/internal/configuration"
	"github.com/odwrtw/polochon/lib"
)

// AppName is the application name
const AppName = "organizer"

// Organizer represents the organizer
type Organizer struct {
	config     *configuration.Config
	videoStore *polochon.VideoStore
	done       chan struct{}
	event      chan string
}

// New returns a new organizer
func New(config *configuration.Config, vs *polochon.VideoStore) *Organizer {
	return &Organizer{
		config:     config,
		videoStore: vs,
		done:       make(chan struct{}),
		event:      make(chan string),
	}
}

// Name returns the name of the app
func (o *Organizer) Name() string {
	return AppName
}

// Run starts the downloader
func (o *Organizer) Run(log *logrus.Entry) error {
	log = log.WithField("app", AppName)

	log.Debug("organizer started")

	// Start the file system notifier
	if err := o.startFsNotifier(log); err != nil {
		return err
	}

	log.Debug("organizer stopped")

	return nil
}

// Stop stops the downloader
func (o *Organizer) Stop(log *logrus.Entry) {
	close(o.done)
}

// startFsNotifier starts the FsNotifier
func (o *Organizer) startFsNotifier(log *logrus.Entry) error {
	var wg sync.WaitGroup

	ctx := polochon.FsNotifierCtx{
		Event: o.event,
		Done:  o.done,
		Wg:    &wg,
	}

	// Launch the FsNotifier
	if err := o.config.Watcher.FsNotifier.Watch(o.config.Watcher.Dir, ctx, log); err != nil {
		return err
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case file := <-ctx.Event:
				log.WithField("event", file).Debugf("got an event")
				if err := o.organize(file, log); err != nil {
					log.Errorf("failed to organize file: %q", err)
				}
			case <-o.done:
				log.Debug("organizer done handling events")
				return
			}
		}
	}()

	// Send a notification to organize the whole folder on app start
	wg.Add(1)
	go func() {
		defer wg.Done()

		log.Info("organize the watched folder")
		ctx.Event <- o.config.Watcher.Dir
	}()

	wg.Wait()

	return nil
}

// Organize stores the videos in the video library
func (o *Organizer) organize(filePath string, log *logrus.Entry) error {
	// Get the file infos from the path
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// If it's a dir we need to walk the dir to organize each file. If it's
	// only a file, organize it.
	if fileInfo.IsDir() {
		err = o.organizeFolder(filePath, log)
	} else {
		err = o.organizeFile(filePath, log)
	}

	return err
}

// OrganizeFile stores the videos in the video library
func (o *Organizer) organizeFile(filePath string, log *logrus.Entry) error {
	log = log.WithField("file_path", filePath)
	log.Debug("organize file")

	// Create a file
	file := polochon.NewFileWithConfig(filePath, o.config.File)

	// Check if file really exists
	if !file.Exists() {
		log.Warning("the file has been removed")
		return nil
	}

	// Check if file is a video
	if !file.IsVideo() {
		log.Debug("the file is not a video")
		return nil
	}

	// Check if file is ignored
	if file.IsIgnored() {
		log.Debug("the file is ignored")
		return nil
	}

	// Check if file is symlink
	if file.IsSymlink() {
		log.Debug("the file is a symlink")
		return nil
	}

	// Check if file is ignored
	if file.IsExcluded() {
		log.Debug("the file is excluded")
		return file.Ignore()
	}

	// Guess the video inforamtion
	video, err := file.Guess(o.config.Movie, o.config.Show, log)
	if err != nil {
		errors.LogErrors(log, err)
		return file.Ignore()
	}

	// Get video details
	if err := video.GetDetails(log); err != nil {
		errors.LogErrors(log, err)
		if errors.IsFatal(err) {
			return file.Ignore()
		}
	}

	// Store the video
	if err := o.videoStore.Add(video, log); err != nil {
		errors.LogErrors(log, err)
		return file.Ignore()
	}

	// Get subtitle
	if err := video.GetSubtitle(log); err != nil {
		errors.LogErrors(log, err)
	}

	// Notify
	o.Notify(video, log)

	return nil
}

// OrganizeFolder organize each file  in a folder
func (o *Organizer) organizeFolder(folderPath string, log *logrus.Entry) error {
	log.WithField("folder_path", folderPath).Debug("organize folder")

	// Walk movies
	err := filepath.Walk(folderPath, func(filePath string, file os.FileInfo, err error) error {
		// Nothing to do on dir
		if file.IsDir() {
			return nil
		}

		// Organize the file
		if err := o.organizeFile(filePath, log); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// Notify sends video to the notifiers
func (o *Organizer) Notify(v polochon.Video, log *logrus.Entry) {
	log = log.WithField("function", "notify")
	for _, n := range o.config.Notifiers {
		if err := n.Notify(v, log); err != nil {
			log.Warnf("failed to send a notification from notifier: %q: %q", n.Name(), err)
		}
	}
}
