package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/odwrtw/polochon/lib"
	"gopkg.in/unrolled/render.v1"
)

// App represents the polochon app
type App struct {
	config *Config

	// Automatic downloader
	downloader *downloader

	// done channel is used to notify all the goroutines to stop
	done chan struct{}
	// stop channel is used to notify the app to stop when the goroutines are
	// properly stopped
	stop chan struct{}
	// errc channel is used to receive errors
	errc chan error

	// wait group sync the goroutines launched by the app
	wg sync.WaitGroup

	videoStore *polochon.VideoStore

	logger *logrus.Logger
	render *render.Render
	mux    *mux.Router
}

// NewApp create a new app from the given configuration path
func NewApp(configPath string) (*App, error) {

	// Setup the logger
	logger := logrus.New()
	logger.Level = logrus.DebugLevel
	logger.Out = os.Stderr
	logger.Formatter = &logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	}

	config, err := LoadConfigFile(configPath, logrus.NewEntry(logger))
	if err != nil {
		log.Panic(err)
	}

	return &App{
		config:     config,
		done:       make(chan struct{}),
		stop:       make(chan struct{}),
		errc:       make(chan error),
		logger:     logger,
		videoStore: polochon.NewVideoStore(config.File, config.Movie, config.Show, logger),
		render:     render.New(),
		mux:        mux.NewRouter(),
	}, nil
}

// Run lauches the app
func (a *App) Run() {
	// Hangle os signals
	osSig := make(chan os.Signal, 1)
	signal.Notify(osSig, os.Interrupt, os.Kill)

	// Handle graceful shutdown
	var forceShutdown bool

	a.logger.Info("Starting app")

	// Start the error logger
	go a.errorLogger()

	// Build the index
	go func() {
		if err := a.videoStore.RebuildIndex(); err != nil {
			a.errc <- err
		}
	}()

	if err := a.startFsNotifier(); err != nil {
		a.logger.Error("Couldn't start the FsNotifier : ", err)
		go a.Stop()
	}

	// Only run the HTTP server if specified
	if a.config.HTTPServer.Enable {
		go a.HTTPServer()
	}

	// Start the downloader
	if a.config.Downloader.Enabled {
		go a.startDownloader()
	}

	for {
		select {
		case <-a.stop:
			a.logger.Info("All done, exiting")
			return
		case <-osSig:
			if forceShutdown {
				a.logger.Warn("Forced shutdown")
				os.Exit(1)
			}
			a.logger.Info("Graceful shutdown")

			// Stop the app
			go a.Stop()

			// Next time it won't be so gentle
			forceShutdown = true
		}
	}
}

// Stop the app
func (a *App) Stop() {
	// Close the done channel to notify every listener
	close(a.done)
	// Wait until they are all done
	a.wg.Wait()
	// Stop the app
	close(a.stop)
}

// startFsNotifier starts the FsNotifier
func (a *App) startFsNotifier() error {
	ctx := polochon.FsNotifierCtx{
		Event: make(chan string),
		Done:  a.done,
		Errc:  a.errc,
		Wg:    &a.wg,
	}

	// Launch the FsNotifier
	if err := a.config.Watcher.FsNotifier.Watch(a.config.Watcher.Dir, ctx, logrus.NewEntry(a.logger)); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case file := <-ctx.Event:
				a.logger.Infof("Got an event for the file: %q", file)
				if err := a.Organize(file); err != nil {
					a.errc <- err
				}
			case <-a.done:
				return
			}
		}
	}()

	// Send a notification to organize the whole folder on app start
	go func() {
		a.logger.Info("Organize the watched folder")
		ctx.Event <- a.config.Watcher.Dir
	}()

	return nil
}

// Run lauches the app
func (a *App) errorLogger() {
	for {
		select {
		case <-a.stop:
			a.logger.Info("Stopping the logger")
			return
		case err := <-a.errc:
			a.logger.Errorf(err.Error())
		}
	}
}

// Organize stores the videos in the video library
func (a *App) Organize(filePath string) error {
	a.wg.Add(1)
	defer a.wg.Done()

	// Logs
	log := a.logger.WithField("function", "organizer")

	// Get the file infos from the path
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// If it's a dir we need to walk the dir to organize each file. If it's
	// only a file, organize it.
	if fileInfo.IsDir() {
		log.Debug("Organize folder")
		err = a.recoverOrganize(a.organizeFolder, filePath, log)
	} else {
		log.Debug("Organize file")
		err = a.recoverOrganize(a.organizeFile, filePath, log)
	}

	if err != nil {
		return err
	}

	return nil
}

// recoverOrganize wraps the organise function to catch panics
func (a *App) recoverOrganize(f func(filePath string, log *logrus.Entry) error, filePath string, log *logrus.Entry) error {
	defer func() {
		if err := recover(); err != nil {
			stack := make([]byte, 8*1024)
			runtime.Stack(stack, true)
			a.logger.Errorf("PANIC: %q\n%s", err, stack)
		}
	}()

	return f(filePath, log)
}

// OrganizeFile stores the videos in the video library
func (a *App) organizeFile(filePath string, log *logrus.Entry) error {
	a.wg.Add(1)
	defer a.wg.Done()

	log = log.WithField("filePath", filePath)

	// Create a file
	file := polochon.NewFileWithConfig(filePath, a.config.File)

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

	// Check if file is ignored
	if file.IsExcluded() {
		log.Debug("the file is excluded")
		return file.Ignore()
	}

	// Guess the video inforamtion
	video, err := file.Guess(a.config.Movie, a.config.Show, log)
	if err != nil {
		log.Errorf("failed to guess video file: %q", err)
		return file.Ignore()
	}

	video.SetLogger(log)

	// Get video details
	ok, merr := video.GetDetails()
	if !ok {
		log.Errorf("failed to get video details: %q", merr)
		for _, err := range merr.Errors {
			log.WithFields(logrus.Fields(err.Ctx)).Debug(err.ErrorStack())
		}
		return file.Ignore()
	}

	// If we already have the video, we delete it to store the new one
	oldVideo, err := a.videoStore.SearchBySlug(video)
	if err != nil {
		// If it's a not and error not found, something's wrong
		if err != polochon.ErrSlugNotFound {
			log.Errorf("SearchBySlug returned an error : %q", err)
		}
	}
	if oldVideo != nil {
		if err := a.videoStore.Delete(oldVideo); err != nil {
			log.Errorf("failed to delete video : %q", err)
		}
	}

	// Store the video
	if err := video.Store(); err != nil {
		log.Errorf("failed to store video: %q", err)
		return file.Ignore()
	}

	// Get subtitle
	if err := video.GetSubtitle(); err != nil {
		log.Errorf("failed to get subtitle")
	}

	// Notify
	if err := video.Notify(); err != nil {
		log.Errorf("failed to notify: %q", err)
	}

	// Rebuild index
	if err := a.videoStore.AddToIndex(video); err != nil {
		log.Errorf("failed to add to index: %q", err)
	}

	return nil
}

// OrganizeFolder organize each file  in a folder
func (a *App) organizeFolder(folderPath string, log *logrus.Entry) error {
	a.wg.Add(1)
	defer a.wg.Done()

	// Walk movies
	err := filepath.Walk(folderPath, func(filePath string, file os.FileInfo, err error) error {
		// Nothing to do on dir
		if file.IsDir() {
			return nil
		}

		// Organize the file
		if err := a.organizeFile(filePath, log); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
