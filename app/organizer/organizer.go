package organizer

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/odwrtw/polochon/app/subapp"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/library"
)

// AppName is the application name
const AppName = "organizer"

// Organizer represents the organizer
type Organizer struct {
	*subapp.Base

	log     *slog.Logger
	config  *configuration.Config
	library *library.Library
	event   chan string
}

// New returns a new organizer
func New(config *configuration.Config, vs *library.Library, log *slog.Logger) *Organizer {
	l := log.With("app", AppName)
	return &Organizer{
		Base:    subapp.NewBase(AppName, l),
		log:     l,
		config:  config,
		library: vs,
	}
}

// Run starts the organizer
func (o *Organizer) Run(ctx context.Context) error {
	// Create the channels
	o.event = make(chan string, 1)
	// Init the app
	o.InitStart()

	defer o.log.Debug("organizer stopped")

	// Start the file system notifier
	return o.startFsNotifier(ctx)
}

// startFsNotifier starts the FsNotifier
func (o *Organizer) startFsNotifier(ctx context.Context) error {
	fsCtx := polochon.FsNotifierCtx{
		Event: o.event,
		Done:  o.Done,
		Wg:    &o.Wg,
	}

	// Send a notification to organize the whole folder on app start
	watcherPath := o.config.Watcher.Dir
	fsCtx.Event <- watcherPath

	// Launch the FsNotifier
	if err := o.config.Watcher.FsNotifier.Watch(watcherPath, fsCtx); err != nil {
		return err
	}

	var err error
	o.Wg.Add(1)
	go func() {
		defer func() {
			o.Wg.Done()
			if r := recover(); r != nil {
				err = subapp.ErrPanicRecovered
				o.Stop()
			}
		}()

		for {
			select {
			case file := <-fsCtx.Event:
				o.log.Debug("got an event", "event", file)
				if err := o.organize(ctx, file); err != nil {
					o.log.Error("failed to organize file", "error", err)
				}
			case <-o.Done:
				o.log.Debug("organizer done handling events")
				return
			}
		}
	}()

	o.Wg.Wait()

	return err
}

// organize stores the videos in the video library
func (o *Organizer) organize(ctx context.Context, filePath string) error {
	// Get the file infos from the path
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// If it's a dir we need to walk the dir to organize each file. If it's
	// only a file, organize it.
	if fileInfo.IsDir() {
		err = o.organizeFolder(ctx, filePath)
	} else {
		err = o.organizeFile(ctx, filePath)
	}

	return err
}

// organizeFile stores the videos in the video library
func (o *Organizer) organizeFile(ctx context.Context, filePath string) error {
	log := o.log.With("file_path", filePath)
	log.Debug("organize file")

	// Create a file
	file := polochon.NewFileWithConfig(filePath, o.config.File)

	// Check if file really exists
	if !file.Exists() {
		log.Warn("the file has been removed")
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

	// Check if file is excluded
	if file.IsExcluded() {
		log.Debug("the file is excluded")
		return file.Ignore()
	}

	// Guess the video information
	video, err := file.Guess(o.config.Movie, o.config.Show, log)
	if err != nil {
		if err != polochon.ErrGuessingVideo {
			log.Error(err.Error())
		}
		return file.Ignore()
	}
	if video == nil {
		log.Error("invalid guess")
		return file.Ignore()
	}

	metadata, err := file.GuessMetadata(log)
	if err != nil {
		log.Warn("failed to guess metadata", "error", err)
	}
	video.SetMetadata(metadata)

	// Get video details
	if err := polochon.GetDetails(ctx, video, log); err != nil {
		if err != polochon.ErrGettingDetails {
			log.Error(err.Error())
		}
		return file.Ignore()
	}

	// Get the video subtitles
	for _, lang := range o.config.SubtitleLanguages {
		_, err := polochon.GetSubtitle(ctx, video, lang, log)
		if err != nil && err != polochon.ErrNoSubtitleFound {
			log.Error(err.Error())
		}
	}

	// Store the video
	if err := o.library.Add(video); err != nil {
		log.Error(err.Error())
		return file.Ignore()
	}

	// Notify
	o.Notify(ctx, video)

	return nil
}

// organizeFolder organizes each file in a folder
func (o *Organizer) organizeFolder(ctx context.Context, folderPath string) error {
	o.log.Debug("organize folder", "folder_path", folderPath)

	// Walk movies
	err := filepath.Walk(folderPath, func(filePath string, file os.FileInfo, err error) error {
		// Nothing to do on dir
		if file.IsDir() {
			return nil
		}

		// Organize the file
		return o.organizeFile(ctx, filePath)
	})

	return err
}

// Notify sends video to the notifiers
func (o *Organizer) Notify(ctx context.Context, v polochon.Video) {
	log := o.log.With("function", "notify")
	for _, n := range o.config.Notifiers {
		if err := n.Notify(ctx, v); err != nil {
			log.Warn("failed to send a notification from notifier", "notifier", n.Name(), "error", err)
		}
	}
}
