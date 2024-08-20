package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var (
	// Default config
	movieDirName     = "movies"
	showDirName      = "tvshows"
	defaultCacheSize = "10MB"
	logLevel         = "info"
	cacheSize        = 10_000_000
	defaultTimeout   = 3 * time.Second
	libraryRefresh   = 1 * time.Minute
	umountLogTimeout = 1 * time.Minute

	// Polochon URL / Token configs
	polochonURL   string
	polochonToken string

	// User rights
	uid, gid uint32
)

func main() {
	if err := run(); err != nil {
		log.Error(err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	pfs, err := newPolochonFs(ctx)
	if err != nil {
		return err
	}

	user, err := user.Current()
	if err != nil {
		return err
	}

	uid64, err := strconv.ParseUint(user.Uid, 10, 32)
	if err != nil {
		return err
	}

	gid64, err := strconv.ParseUint(user.Gid, 10, 32)
	if err != nil {
		return err
	}

	flag.StringVar(&pfs.mountPoint, "mountPoint", "", "path to mount the filesystem")
	flag.StringVar(&polochonURL, "url", os.Getenv("POLOCHON_URL"), "polochon API URL")
	flag.StringVar(&polochonToken, "token", os.Getenv("POLOCHON_TOKEN"), "polochon API token")
	flag.StringVar(&showDirName, "showDirName", showDirName, "show directory name")
	flag.StringVar(&movieDirName, "movieDirName", movieDirName, "movie directory name")
	flag.StringVar(&defaultCacheSize, "cache", defaultCacheSize, "cache size (e.g. 10MB)")
	flag.StringVar(&logLevel, "logLevel", logLevel, "log level (warn, error, info, debug, trace)")
	flag.DurationVar(&defaultTimeout, "timeout", defaultTimeout, "HTTP requests timeout")
	flag.DurationVar(&libraryRefresh, "libraryRefresh", libraryRefresh, "library refresh timer")
	flag.Uint64Var(&uid64, "uid", uid64, "UID of the mounted files")
	flag.Uint64Var(&gid64, "gid", uid64, "GID of the mounted files")
	flag.BoolVar(&pfs.fuseDebug, "fuseDebug", false, "debug fuse events")
	flag.Parse()

	uid = uint32(uid64)
	gid = uint32(gid64)

	cacheBytes, err := humanize.ParseBytes(defaultCacheSize)
	if err != nil {
		return err
	}
	cacheSize = int(cacheBytes)

	// Setup logger
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:05",
	})

	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	log.SetOutput(os.Stdout)
	log.SetLevel(lvl)

	err = pfs.init()
	if err != nil {
		flag.PrintDefaults()
		return err
	}

	log.WithField("mountpoint", pfs.mountPoint).Info("Mouting polochonfs")
	server, err := pfs.mount()
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	go func() {
		wg.Add(1)
		defer wg.Done()
		pfs.handleUpdates()
	}()

	log.Info("FUSE daemon started")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	cancel()

	wg.Wait()
	pfs.unmount(server)
	return nil
}
