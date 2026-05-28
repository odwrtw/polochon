package organizer

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/library"
)

// AppName is the application name
const AppName = "organizer"

// Organizer represents the organizer
type Organizer struct {
	log     *slog.Logger
	config  *configuration.Config
	library *library.Library
}

// New returns a new organizer
func New(config *configuration.Config, vs *library.Library, log *slog.Logger) *Organizer {
	return &Organizer{
		log:     log.With("app", AppName),
		config:  config,
		library: vs,
	}
}

// Run starts the organizer
func (o *Organizer) Run(ctx context.Context) error {
	event := make(chan string, 1)
	event <- o.config.Watcher.Dir

	var wg sync.WaitGroup
	fsCtx := polochon.FsNotifierCtx{
		Event: event,
		Done:  ctx.Done(),
		Wg:    &wg,
	}

	if err := o.config.Watcher.FsNotifier.Watch(o.config.Watcher.Dir, fsCtx); err != nil {
		return err
	}

	for {
		select {
		case file := <-event:
			o.log.Debug("got an event", "event", file)
			if err := o.organize(ctx, file); err != nil {
				o.log.Error("failed to organize file", "error", err)
			}
		case <-ctx.Done():
			wg.Wait()
			o.log.Debug("organizer stopped")
			return nil
		}
	}
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
	err := filepath.WalkDir(folderPath, func(filePath string, file os.DirEntry, err error) error {
		if err != nil {
			return err
		}

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
