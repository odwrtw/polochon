package app

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/app/internal/cleaner"
	"github.com/odwrtw/polochon/app/internal/configuration"
	"github.com/odwrtw/polochon/app/internal/downloader"
	"github.com/odwrtw/polochon/app/internal/organizer"
	"github.com/odwrtw/polochon/app/internal/server"
	"github.com/odwrtw/polochon/app/internal/token"
	"github.com/odwrtw/polochon/lib"
)

// SubApp represents an application launched by the App
type SubApp interface {
	// Name returns the name of the sub app
	Name() string
	// Run starts the SubApp, it should be a synchronous call
	Run(log *logrus.Entry) error
	// Stop sends a signal to the SubApp to stop gracefully, this should be an
	// asynchronous call
	Stop(log *logrus.Entry)
}

// App represents the polochon app
type App struct {
	// Keep the config file paths to be able to reload the app later
	configPath      string
	tokenConfigPath string

	// subApps hold the sub applications
	subApps []SubApp

	// done is channel used to stop the app
	done chan struct{}

	// wait group sync the goroutines launched by the app
	wg sync.WaitGroup

	// app logger
	logger *logrus.Logger
}

// NewApp create a new app from the given configuration path
func NewApp(configPath, tokenManagerPath string) (*App, error) {
	// Setup the logger
	logger := logrus.New()
	logger.Level = logrus.DebugLevel
	logger.Out = os.Stderr
	logger.Formatter = &logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	}

	// Create the app
	app := &App{
		configPath:      configPath,
		tokenConfigPath: tokenManagerPath,
		done:            make(chan struct{}),
		logger:          logger,
	}

	// Init the app
	if err := app.init(); err != nil {
		return nil, err
	}

	return app, nil
}

// init the app by reading the configuration files
func (a *App) init() error {
	log := logrus.NewEntry(a.logger).WithField("function", "app_init")
	log.Debug("loading app configuration")

	config, err := configuration.LoadConfigFile(a.configPath, log)
	if err != nil {
		return err
	}

	videoStore := polochon.NewVideoStore(config.File, config.Movie, config.Show, config.VideoStore)

	// Build the videoStore index
	if err := videoStore.RebuildIndex(log); err != nil {
		log.WithField("function", "rebuild_index").Error(err)
	}

	// Add the organizer
	a.subApps = []SubApp{organizer.New(config, videoStore)}

	if config.Downloader.Enabled {
		// Add the downloader
		a.subApps = append(a.subApps, downloader.New(config, videoStore))

		if config.Downloader.Cleaner.Enabled {
			// Add the cleaner
			a.subApps = append(a.subApps, cleaner.New(config))
		}
	}

	// Only run the HTTP server if specified
	if config.HTTPServer.Enable {
		// Read the config of the token manager
		var tokenManager *token.Manager
		if _, err := os.Stat(a.tokenConfigPath); err == nil {
			log.Debug("loading token manager configuration")

			file, err := os.Open(a.tokenConfigPath)
			defer file.Close()
			if err != nil {
				return err
			}

			tokenManager, err = token.LoadFromYaml(file)
			if err != nil {
				return err
			}
			log.Debug("token manager configuration loaded")
		}

		// Add the http server
		a.subApps = append(a.subApps, server.New(config, videoStore, tokenManager))
	}

	log.Debug("app configuration loaded")

	return nil
}

// Run launches the app
func (a *App) Run() {
	// Hangle os signals
	osSig := make(chan os.Signal, 1)
	signal.Notify(osSig, syscall.SIGINT, syscall.SIGKILL, syscall.SIGHUP)

	log := logrus.NewEntry(a.logger)
	a.logger.Info("starting the app")

	// Start all the apps
	a.startApps(log)

	// Handle graceful shutdown
	var forceShutdown bool

	for {
		select {
		case <-a.done:
			a.logger.Info("all done, exiting")
			return
		case sig := <-osSig:
			log.WithField("os_event", sig).Info("got an os event")
			switch sig {
			case syscall.SIGINT, syscall.SIGKILL:
				if forceShutdown {
					log.Warn("forced shutdown")
					os.Exit(1)
				}
				log.Info("graceful shutdown")

				// stop the app
				go a.Stop(log)

				// Next time it won't be so gentle
				forceShutdown = true

			case syscall.SIGHUP:
				log.Info("reloading app")

				a.stopApps(log)

				if err := a.init(); err != nil {
					log.Fatal(err)
				}

				a.startApps(log)

				log.Info("app reloaded")
			}
		}
	}
}

// startApps launches all the sub app in their own goroutine
func (a *App) startApps(log *logrus.Entry) {
	log.Debug("starting the sub apps")
	for _, subApp := range a.subApps {
		log.Debugf("starting sub app %q", subApp.Name())
		a.wg.Add(1)
		go func(app SubApp) {
			defer a.wg.Done()
			if err := app.Run(log); err != nil {
				log.Error(err)
			}
		}(subApp)
	}
}

// stopApps stops all the sub apps
func (a *App) stopApps(log *logrus.Entry) {
	log.Debug("stopping the sub apps")
	for _, subApp := range a.subApps {
		log.Debugf("stopping sub app %q", subApp.Name())
		subApp.Stop(log)
	}

	a.wg.Wait()
	log.Debug("sub apps stopped gracefully")
}

// Stop stops the app
func (a *App) Stop(log *logrus.Entry) {
	a.stopApps(log)
	close(a.done)
}
