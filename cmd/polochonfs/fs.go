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
)

type polochonfs struct {
	debug bool
	ctx   context.Context

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
	fmt.Println("Adding persistent nodes")
	pfs.root.addChild(newPersistentNodeDir(movieDirName))
	pfs.root.addChild(newPersistentNodeDir(showDirName))

	pfs.updateMovies(ctx)
	pfs.updateShows(ctx)

	fmt.Println("All done")
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
				fmt.Println("Updating movies from signal")
				pfs.updateMovies(pfs.ctx)
			case syscall.SIGUSR2:
				fmt.Println("Updating shows from signal")
				pfs.updateShows(pfs.ctx)
			}
		case <-ticker.C:
			fmt.Println("Handle updates from ticker")
			pfs.updateMovies(pfs.ctx)
			pfs.updateShows(pfs.ctx)
		case <-pfs.ctx.Done():
			fmt.Println("Handle updates done")
			return
		}
	}
}

func (pfs *polochonfs) mount() (*fuse.Server, error) {
	return fs.Mount(pfs.mountPoint, pfs.root, &fs.Options{
		MountOptions: fuse.MountOptions{
			Options: []string{"ro"},
			Name:    "polochonfs",
			FsName:  "polochonfs",
			Debug:   pfs.debug,
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

	fmt.Println("Waiting for unmount...")
	start := time.Now()
	for {
		err = server.Unmount()
		if err == nil {
			break
		}

		if time.Since(start) > umountLogTimeout {
			fmt.Println("Still waiting for unmount...")
			start = time.Now()
		}

		fmt.Println("Failed to unmount: ", err)
		time.Sleep(1 * time.Second)
	}

	fmt.Println("Unmount successfull")
}
