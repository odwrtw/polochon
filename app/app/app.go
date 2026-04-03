package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/odwrtw/polochon/app/auth"
	"github.com/odwrtw/polochon/app/dm"
	"github.com/odwrtw/polochon/app/downloader"
	"github.com/odwrtw/polochon/app/organizer"
	"github.com/odwrtw/polochon/app/safeguard"
	"github.com/odwrtw/polochon/app/server"
	"github.com/odwrtw/polochon/app/subapp"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/library"
)

// App represents the polochon app
type App struct {
	// Keep the config file paths to be able to reload the app later
	configPath     string
	authConfigPath string

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
	log *slog.Logger
}

// NewApp create a new app from the given configuration path
func NewApp(configPath, authManagerPath string) (*App, error) {
	// Create the app
	app := &App{
		configPath:     configPath,
		authConfigPath: authManagerPath,
		safeguard:      safeguard.New(),
		done:           make(chan struct{}),
		reload:         make(chan subapp.App),
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
	a.log = config.Logger

	log := a.log.With("function", "app_init")
	log.Debug("app configuration loaded")

	lib := library.New(config)

	// Build the library index
	if err := lib.RebuildIndex(); err != nil {
		log.With("function", "rebuild_index").Error(err.Error())
	}

	a.subApps = []subapp.App{}
	if config.Organizer.Enabled {
		// Add the organizer
		a.subApps = append(a.subApps, organizer.New(config, lib, a.log))
	}

	if config.Downloader.Enabled {
		// Add the downloader
		a.subApps = append(a.subApps, downloader.New(config, lib, a.log))
	}

	if config.DownloadManager.Enabled {
		// Add the download manager
		a.subApps = append(a.subApps, dm.New(config, lib, a.log))
	}

	// Only run the HTTP server if specified
	if config.HTTPServer.Enable {
		// Read the config of the auth manager
		var authManager *auth.Manager
		if _, err := os.Stat(a.authConfigPath); err == nil {
			log.Debug("loading auth manager configuration")

			file, err := os.Open(a.authConfigPath)
			if err != nil {
				return err
			}
			defer func() { _ = file.Close() }()

			authManager, err = auth.New(file)
			if err != nil {
				return err
			}
			log.Debug("auth manager configuration loaded")
		}

		// Add the http server
		srv := server.New(config, lib, authManager, a.log)
		config.Notifiers = append(config.Notifiers, srv.Hub())
		a.subApps = append(a.subApps, srv)
	}

	log.Debug("app configuration loaded")

	return nil
}

// Run launches the app
func (a *App) Run() {
	// Handle os signals
	osSig := make(chan os.Signal, 1)
	signal.Notify(osSig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	a.log.Info("starting the app")

	// Panic loop safeguard
	go func() {
		if err := a.safeguard.Run(a.log); err != nil {
			a.log.Error(err.Error())
			go a.Stop()
		}
	}()

	// Start all the apps
	a.startSubApps()

	// Handle graceful shutdown
	var forceShutdown bool

	// Main loop
	for {
		select {
		case <-a.done:
			a.log.Info("all done, exiting")
			return
		case subApp := <-a.reload:
			a.log.Info("reloading sub app", "app", subApp.Name())
			a.wg.Go(func() {
				subApp.BlockingStop()
				a.subAppStart(subApp)
			})
		case sig := <-osSig:
			a.log.Info("got an os event", "os_event", sig)
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				if forceShutdown {
					a.log.Warn("forced shutdown")
					os.Exit(1)
				}
				a.log.Info("graceful shutdown")

				// stop the app
				go a.Stop()

				// Next time it won't be so gentle
				forceShutdown = true

			case syscall.SIGHUP:
				a.log.Info("reloading app")

				a.stopApps()

				if err := a.init(); err != nil {
					a.log.Error(err.Error())
					os.Exit(1)
				}

				a.startSubApps()

				a.log.Info("app reloaded")
			}
		}
	}
}

// startSubApps launches all the sub app
func (a *App) startSubApps() {
	a.log.Debug("starting the sub apps")
	for _, subApp := range a.subApps {
		a.subAppStart(subApp)
	}
}

// stopApps stops all the sub apps
func (a *App) stopApps() {
	a.log.Debug("stopping the sub apps")
	for _, subApp := range a.subApps {
		a.log.Debug("stopping sub app", "app", subApp.Name())
		subApp.Stop()
	}

	a.wg.Wait()
	a.log.Debug("sub apps stopped gracefully")
}

// Stop stops the app
func (a *App) Stop() {
	a.stopApps()
	a.safeguard.BlockingStop()
	close(a.done)
}

// subAppStart starts a sub app in its own goroutine
func (a *App) subAppStart(app subapp.App) {
	a.log.Debug("starting sub app", "app", app.Name())
	ctx := context.Background()
	a.wg.Go(func() {
		if err := app.Run(ctx); err != nil {
			// Check the error, if it comes from a panic recovery reload the
			// app
			if err == subapp.ErrPanicRecovered {
				a.log.Error(err.Error(), "app", app.Name())

				// Notify the safeguard of the error
				a.safeguard.Event()

				// Write to the reload channel in a goroutine to prevent deadlocks
				go func() {
					a.reload <- app
				}()
			} else {
				// Only log the error
				a.log.Error(err.Error())
				go a.Stop()
			}
		}
	})
	a.log.Debug("sub app started", "app", app.Name())
}
