package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"

	"github.com/Sirupsen/logrus"
	"gitlab.quimbo.fr/odwrtw/polochon/lib"
)

// App represents the polochon app
type App struct {
	config *polochon.Config
	ctx    *polochon.FsNotifierCtx

	// event channel is used to trigger the organize a file
	event chan string
	// done channel is used to notify all the goroutines to stop
	done chan struct{}
	// stop channel is used to notify the app to stop when the goroutines are
	// properly stopped
	stop chan struct{}
	// errc channel is used to receive errors
	errc chan error

	// wait group sync the goroutines launched by the app
	wg sync.WaitGroup
}

// NewApp create a new app from the given configuration path
func NewApp(configPath string) (*App, error) {
	config, err := polochon.ReadConfigFile(configPath)
	if err != nil {
		log.Panic(err)
	}

	err = config.Init()
	if err != nil {
		log.Panic(err)
	}

	return &App{
		config: config,
		event:  make(chan string),
		done:   make(chan struct{}),
		stop:   make(chan struct{}),
		errc:   make(chan error),
	}, nil
}

// Run lauches the app
func (a *App) Run() {
	// Hangle os signals
	osSig := make(chan os.Signal, 1)
	signal.Notify(osSig, os.Interrupt, os.Kill)

	// Handle graceful shutdown
	var forceShutdown bool

	a.config.Log.Info("Starting app")

	// Start the error logger
	go a.errorLogger()

	if err := a.StartFsNotifier(); err != nil {
		a.config.Log.Error("Couldn't start the FsNotifier : ", err)
		go a.Stop()
	}

	// Only run the HTTP server if specified
	if a.config.HTTPServer.Enable {
		go a.HTTPServer()
	}

	for {
		select {
		case <-a.stop:
			a.config.Log.Info("All done, exiting")
			return
		case <-osSig:
			if forceShutdown {
				a.config.Log.Info("Forced shutdown")
				os.Exit(1)
			}
			a.config.Log.Info("Graceful shutdown")

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

// StartFsNotifier starts the FsNotifier
func (a *App) StartFsNotifier() error {
	ctx := polochon.FsNotifierCtx{
		Event: a.event,
		Done:  a.done,
		Errc:  a.errc,
		Wg:    &a.wg,
	}

	// Launch the FsNotifier
	if err := a.config.Watcher.FsNotifier.Watch(a.config.Watcher.Dir, ctx); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case file := <-a.event:
				a.config.Log.Infof("Got an event for the file: %q", file)
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
		a.config.Log.Info("Organize the watched folder")
		a.event <- a.config.Watcher.Dir
	}()

	return nil
}

// Run lauches the app
func (a *App) errorLogger() {
	for {
		select {
		case <-a.stop:
			a.config.Log.Info("Stopping the logger")
			return
		case err := <-a.errc:
			a.config.Log.Errorf("Err: %q", err)
		}
	}
}

// Organize stores the videos in the video library
func (a *App) Organize(filePath string) error {
	a.wg.Add(1)
	defer a.wg.Done()

	// Logs
	log := a.config.Log.WithFields(logrus.Fields{
		"filepath": filePath,
		"function": "organizer",
	})

	// Get the file infos from the path
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// If it's a dir we need to walk the dir to organize each file. If it's
	// only a file, organize it.
	if fileInfo.IsDir() {
		log.Debug("Organize folder")
		err = a.organizeFolder(filePath, log)
	} else {
		log.Debug("Organize file")
		err = a.organizeFile(filePath, log)
	}

	if err != nil {
		return err
	}

	return nil
}

// OrganizeFile stores the videos in the video library
func (a *App) organizeFile(filePath string, log *logrus.Entry) error {
	a.wg.Add(1)
	defer a.wg.Done()

	log = log.WithField("filePath", filePath)

	// Create a file
	file := polochon.NewFileWithConfig(filePath, a.config)

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
	video, err := file.Guess()
	if err != nil {
		log.Errorf("failed to guess video file: %q", err)
		return file.Ignore()
	}

	video.SetConfig(a.config)

	// Get video details
	if err := video.GetDetails(); err != nil {
		log.Errorf("failed to get video details: %q", err)
		return file.Ignore()
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
