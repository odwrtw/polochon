package organizer

import (
	"os"
	"path/filepath"

	"github.com/odwrtw/polochon/app/subapp"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/library"
	"github.com/sirupsen/logrus"
)

// AppName is the application name
const AppName = "organizer"

// Organizer represents the organizer
type Organizer struct {
	*subapp.Base

	config  *configuration.Config
	library *library.Library
	event   chan string
}

// New returns a new organizer
func New(config *configuration.Config, vs *library.Library) *Organizer {
	return &Organizer{
		Base:    subapp.NewBase(AppName),
		config:  config,
		library: vs,
	}
}

// Run starts the downloader
func (o *Organizer) Run(log *logrus.Entry) error {
	// Create the channels
	o.event = make(chan string, 1)
	// Init the app
	o.InitStart(log)

	log = log.WithField("app", AppName)

	defer log.Debug("organizer stopped")

	// Start the file system notifier
	return o.startFsNotifier(log)
}

// startFsNotifier starts the FsNotifier
func (o *Organizer) startFsNotifier(log *logrus.Entry) error {
	ctx := polochon.FsNotifierCtx{
		Event: o.event,
		Done:  o.Done,
		Wg:    &o.Wg,
	}

	// Send a notification to organize the whole folder on app start
	watcherPath := o.config.Watcher.Dir
	ctx.Event <- watcherPath

	// Launch the FsNotifier
	if err := o.config.Watcher.FsNotifier.Watch(watcherPath, ctx, log); err != nil {
		return err
	}

	var err error
	o.Wg.Add(1)
	go func() {
		defer func() {
			o.Wg.Done()
			if r := recover(); r != nil {
				err = subapp.ErrPanicRecovered
				o.Stop(log)
			}
		}()

		for {
			select {
			case file := <-ctx.Event:
				log.WithField("event", file).Debugf("got an event")
				if err := o.organize(file, log); err != nil {
					log.Errorf("failed to organize file: %q", err)
				}
			case <-o.Done:
				log.Debug("organizer done handling events")
				return
			}
		}
	}()

	o.Wg.Wait()

	return err
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
		if err != polochon.ErrGuessingVideo {
			log.Error(err)
		}
		return file.Ignore()
	}
	if video == nil {
		log.Error("invalid guess")
		return file.Ignore()
	}

	metadata, err := file.GuessMetadata(log)
	if err != nil {
		log.Warnf("failed to guess metadata: %s", err.Error())
	}
	video.SetMetadata(metadata)

	// Get video details
	if err := polochon.GetDetails(video, log); err != nil {
		if err != polochon.ErrGettingDetails {
			log.Error(err)
		}
		return file.Ignore()
	}

	// Get the video subtitles
	for _, lang := range o.config.SubtitleLanguages {
		_, err := polochon.GetSubtitle(video, lang, log)
		if err != nil && err != polochon.ErrNoSubtitleFound {
			log.Error(err)
		}
	}

	// Store the video
	if err := o.library.Add(video, log); err != nil {
		log.Error(err)
		return file.Ignore()
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
		return o.organizeFile(filePath, log)
	})

	return err
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
