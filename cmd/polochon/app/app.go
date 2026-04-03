package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/odwrtw/polochon/cmd/polochon/dm"
	"github.com/odwrtw/polochon/cmd/polochon/downloader"
	"github.com/odwrtw/polochon/cmd/polochon/organizer"
	"github.com/odwrtw/polochon/cmd/polochon/server"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/library"
)

type subapp struct {
	name string
	run  func(ctx context.Context) error
}

// App represents the polochon app
type App struct {
	// Keep the config file paths to be able to reload the app later
	configPath     string
	authConfigPath string

	subapps []subapp

	log *slog.Logger
}

// NewApp create a new app from the given configuration path
func NewApp(configPath, authManagerPath string) (*App, error) {
	app := &App{
		configPath:     configPath,
		authConfigPath: authManagerPath,
	}

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

	a.subapps = nil

	if config.Organizer.Enabled {
		org := organizer.New(config, lib, a.log)
		a.subapps = append(a.subapps, subapp{name: "organizer", run: org.Run})
	}

	if config.Downloader.Enabled {
		dl := downloader.New(config, lib, a.log)
		a.subapps = append(a.subapps, subapp{name: "downloader", run: dl.Run})
	}

	if config.DownloadManager.Enabled {
		dlm := dm.New(config, lib, a.log)
		a.subapps = append(a.subapps, subapp{name: "download_manager", run: dlm.Run})
	}

	if config.HTTPServer.Enable {
		srv, err := server.New(config, lib, a.authConfigPath, a.log)
		if err != nil {
			return err
		}
		config.Notifiers = append(config.Notifiers, srv.Hub())
		a.subapps = append(a.subapps, subapp{name: "http_server", run: srv.Run})
	}

	return nil
}

// Run launches the app
func (a *App) Run() {
	a.log.Info("starting the app")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, gctx := errgroup.WithContext(ctx)
	for _, sub := range a.subapps {
		g.Go(func() error { return a.runOne(gctx, sub) })
	}
	g.Go(func() error { return a.handleSignals(gctx, cancel) })

	if err := g.Wait(); err != nil {
		a.log.Error(err.Error())
	}
}

func (a *App) runOne(ctx context.Context, sub subapp) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s: panic: %v", sub.name, r)
		}
	}()
	a.log.Info("starting subapp", "app", sub.name)
	err = sub.run(ctx)
	if err != nil {
		a.log.Error("subapp stopped with error", "app", sub.name, "error", err)
	}
	return err
}

func (a *App) handleSignals(ctx context.Context, cancel context.CancelFunc) error {
	osSig := make(chan os.Signal, 1)
	signal.Notify(osSig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(osSig)

	var forceShutdown bool
	for {
		select {
		case <-ctx.Done():
			return nil
		case sig := <-osSig:
			if forceShutdown {
				a.log.Warn("forced shutdown")
				os.Exit(1)
			}
			a.log.Info("graceful shutdown", "signal", sig)
			cancel()
			forceShutdown = true
		}
	}
}
