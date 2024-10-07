package main

import (
	"context"
	"fmt"
	logger "log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/odwrtw/polochon/lib/papi"
	log "github.com/sirupsen/logrus"
)

var (
	baseInode   uint64 = 0xFFFFFFFF00000000
	movieInodes        = map[uint64]struct{}{}
	showInodes         = map[uint64]struct{}{}
)

type polochonfs struct {
	fuseDebug bool
	ctx       context.Context
	wg        sync.WaitGroup

	root *node

	mountPoint string
	client     *papi.Client
}

func newPolochonFs(ctx context.Context) (*polochonfs, error) {
	return &polochonfs{
		ctx:  ctx,
		root: newRootNode(),
	}, nil
}

func (pfs *polochonfs) init() error {
	for _, field := range []struct {
		value, errMsg string
	}{
		{value: pfs.mountPoint, errMsg: "missing mountpoint"},
		{value: polochonURL, errMsg: "missing url"},
		{value: polochonToken, errMsg: "missing token"},
	} {
		if field.value == "" {
			return fmt.Errorf("%s", field.errMsg)
		}
	}

	pfs.wg = sync.WaitGroup{}

	var err error
	pfs.client, err = papi.New(polochonURL)
	if err != nil {
		return err
	}
	pfs.client.SetToken(polochonToken)
	pfs.client.SetTimeout(defaultTimeout)

	return nil
}

func (pfs *polochonfs) wait() {
	pfs.wg.Wait()
}

func (pfs *polochonfs) buildFS(_ context.Context) {
	pfs.root.addChild(newNodeDir(movieDirName, movieDirName, pfs.root.times))
	pfs.root.addChild(newNodeDir(showDirName, showDirName, pfs.root.times))

	go pfs.handleUpdates()
}

func (pfs *polochonfs) handleUpdates() {
	pfs.wg.Add(1)
	defer pfs.wg.Done()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR1, syscall.SIGUSR2)

	pfs.updateMovies()
	pfs.updateShows()

	ticker := time.NewTicker(libraryRefresh)
	defer ticker.Stop()

	for {
		select {
		case s := <-sigs:
			switch s {
			case syscall.SIGUSR1:
				log.Info("Updating movies from signal")
				ticker.Reset(libraryRefresh)
				pfs.updateMovies()
			case syscall.SIGUSR2:
				log.Info("Updating shows from signal")
				ticker.Reset(libraryRefresh)
				pfs.updateShows()
			}
		case <-ticker.C:
			log.Debug("Handle updates from ticker")
			pfs.updateMovies()
			pfs.updateShows()
		case <-pfs.ctx.Done():
			log.Info("Handle updates done")
			return
		}
	}
}

func (pfs *polochonfs) mount() (*fuse.Server, error) {
	customLogger := logger.New(os.Stdout,
		"\033[33mFUSE\033[0m ", // Yellow prefix
		logger.Ldate|logger.Ltime,
	)
	return fs.Mount(pfs.mountPoint, pfs.root, &fs.Options{
		RootStableAttr: &fs.StableAttr{Ino: baseInode},
		UID:            uid,
		GID:            gid,
		Logger:         customLogger,
		MountOptions: fuse.MountOptions{
			// Enforce sequential read, one read at a time. This is useful to
			// read from the http body directly.
			SyncRead: true,
			// Mount in read only mode
			Options: []string{"ro"},
			Name:    "polochonfs",
			FsName:  "polochonfs",
			Debug:   pfs.fuseDebug,
			Logger:  customLogger,
		},
		OnAdd: pfs.buildFS,
	})
}

func (pfs *polochonfs) unmount(server *fuse.Server) {
	var err error

	log.Info("Waiting for unmount...")
	start := time.Now()
	for {
		err = server.Unmount()
		if err == nil {
			break
		}

		if time.Since(start) > umountLogTimeout {
			log.Debug("Still waiting for unmount...")
			start = time.Now()
		}

		log.WithField("error", err).Error("Failed to unmount")
		time.Sleep(1 * time.Second)
	}

	log.Info("polochonfs unmounted successfully")
}
