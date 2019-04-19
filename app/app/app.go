package app

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/odwrtw/errors"
	"github.com/odwrtw/polochon/app/cleaner"
	"github.com/odwrtw/polochon/app/downloader"
	"github.com/odwrtw/polochon/app/organizer"
	"github.com/odwrtw/polochon/app/safeguard"
	"github.com/odwrtw/polochon/app/server"
	"github.com/odwrtw/polochon/app/subapp"
	"github.com/odwrtw/polochon/app/token"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/library"
	"github.com/sirupsen/logrus"
)

// App represents the polochon app
type App struct {
	// Keep the config file paths to be able to reload the app later
	configPath      string
	tokenConfigPath string

	// subApps hold the sub applications
	subApps []subapp.App

	// done is channel used to stop the app
	done chan struct{}

	// safeguard
	safeguard *safeguard.Safeguard

	reload chan subapp.App

	// wait group sync the goroutines launched by the app
	wg sync.WaitGroup

	// app logger
	logger *logrus.Logger
}

// NewApp create a new app from the given configuration path
func NewApp(configPath, tokenManagerPath string) (*App, error) {
	// Create the app
	app := &App{
		configPath:      configPath,
		tokenConfigPath: tokenManagerPath,
		safeguard:       safeguard.New(),
		done:            make(chan struct{}),
		reload:          make(chan subapp.App),
	}

	// Init the app
	if err := app.init(); err != nil {
		return nil, err
	}

	return app, nil
}

// init the app by reading the configuration files
func (a *App) init() error {
	config, err := configuration.LoadConfigFile(a.configPath)
	if err != nil {
		return err
	}
	a.logger = config.Logger

	log := logrus.NewEntry(a.logger).WithField("function", "app_init")
	log.Debug("app configuration loaded")

	library := library.New(config)

	// Build the library index
	if err := library.RebuildIndex(log); err != nil {
		log.WithField("function", "rebuild_index").Error(err)
	}

	// Add the organizer
	a.subApps = []subapp.App{organizer.New(config, library)}

	if config.Downloader.Enabled {
		// Add the downloader
		a.subApps = append(a.subApps, downloader.New(config, library))

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
		a.subApps = append(a.subApps, server.New(config, library, tokenManager))
	}

	log.Debug("app configuration loaded")

	return nil
}

// Run launches the app
func (a *App) Run() {
	// Hangle os signals
	osSig := make(chan os.Signal, 1)
	signal.Notify(osSig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	log := logrus.NewEntry(a.logger)
	log.Info("starting the app")

	// Panic loop safeguard
	go func() {
		if err := a.safeguard.Run(log); err != nil {
			errors.LogErrors(log, err)
			go a.Stop(log)
		}
	}()

	// Start all the apps
	a.startSubApps(log)

	// Handle graceful shutdown
	var forceShutdown bool

	// Main loop
	for {
		select {
		case <-a.done:
			log.Info("all done, exiting")
			return
		case subApp := <-a.reload:
			log.Infof("reloading sub app %q", subApp.Name())
			a.wg.Add(1)
			go func() {
				defer a.wg.Done()
				subApp.BlockingStop(log)
				a.subAppStart(subApp, log)
			}()
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

				a.startSubApps(log)

				log.Info("app reloaded")
			}
		}
	}
}

// startSubApps launches all the sub app
func (a *App) startSubApps(log *logrus.Entry) {
	log.Debug("starting the sub apps")
	for _, subApp := range a.subApps {
		a.subAppStart(subApp, log)
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
	a.safeguard.BlockingStop(log)
	close(a.done)
}

// Start statrs a sub app in its own goroutine
func (a *App) subAppStart(app subapp.App, log *logrus.Entry) {
	log.Debugf("starting sub app %q", app.Name())
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()

		if err := app.Run(log); err != nil {
			// Check the error type, if it comes from a panic recovery
			// reload the app
			switch e := err.(type) {
			case *errors.Error:
				errors.LogErrors(log.WithField("app", app.Name()), e)

				// Notify the safeguard of the error
				a.safeguard.Event()

				// Write to the reload channel in a goroutine to prevent deadlocks
				go func() {
					a.reload <- app
				}()
			// Only log the error
			default:
				log.Error(err)
				go a.Stop(log)
			}
		}
	}()
	log.Debugf("sub app %q started", app.Name())
}
