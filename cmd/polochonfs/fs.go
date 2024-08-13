package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/odwrtw/polochon/lib/papi"
	log "github.com/sirupsen/logrus"
)

var (
	movieInodes = map[uint64]struct{}{}
	showInodes  = map[uint64]struct{}{}
)

type polochonfs struct {
	debug, fuseDebug bool
	ctx              context.Context

	root *node

	mountPoint string
	client     *papi.Client
}

func newPolochonFs(ctx context.Context) (*polochonfs, error) {
	return &polochonfs{
		ctx:   ctx,
		root:  newRootNode(),
		debug: true,
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
			return fmt.Errorf(field.errMsg)
		}
	}

	var err error
	pfs.client, err = papi.New(polochonURL)
	if err != nil {
		return err
	}
	pfs.client.SetToken(polochonToken)
	pfs.client.SetTimeout(httpTimeout)

	return nil
}

func (pfs *polochonfs) buildFS(ctx context.Context) {
	log.Debug("Adding persistent nodes\n")
	pfs.root.addChild(newPersistentNodeDir(movieDirName))
	pfs.root.addChild(newPersistentNodeDir(showDirName))

	pfs.updateMovies(ctx)
	pfs.updateShows(ctx)
}

func (pfs *polochonfs) handleUpdates() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR1, syscall.SIGUSR2)

	ticker := time.NewTicker(libraryRefresh)
	defer ticker.Stop()

	for {
		select {
		case s := <-sigs:
			switch s {
			case syscall.SIGUSR1:
				log.Debug("Updating movies from signal")
				pfs.updateMovies(pfs.ctx)
			case syscall.SIGUSR2:
				log.Debug("Updating shows from signal")
				pfs.updateShows(pfs.ctx)
			}
		case <-ticker.C:
			log.Debug("Handle updates from ticker")
			pfs.updateMovies(pfs.ctx)
			pfs.updateShows(pfs.ctx)
		case <-pfs.ctx.Done():
			log.Debug("Handle updates done")
			return
		}
	}
}

func (pfs *polochonfs) mount() (*fuse.Server, error) {
	return fs.Mount(pfs.mountPoint, pfs.root, &fs.Options{
		MountOptions: fuse.MountOptions{
			Options: []string{"ro"},
			Name:    "polochon",
			FsName:  "polochonfs",
			Debug:   pfs.fuseDebug,
			// Try to enable direct mount to avoid fusermount
			// DirectMount: true,
			// Disable extended attributes
			DisableXAttrs: true,
			// FUSE will wait for one callback to return before calling another
			SingleThreaded: true,
			// DisableReadDirPlus: true,
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

		log.Error("Failed to unmount: ", err)
		time.Sleep(1 * time.Second)
	}

	log.Info("polochonfs unmounted successfully")
}
