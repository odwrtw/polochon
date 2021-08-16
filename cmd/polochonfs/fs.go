package main

import (
	"context"
	"fmt"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/odwrtw/polochon/lib/papi"
)

type polochonfs struct {
	debug bool
	ctx   context.Context

	root *node

	movies []*papi.Movie
	shows  []*papi.Show

	mountPoint, url, token string
	client                 *papi.Client
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
		{value: pfs.url, errMsg: "missing url"},
		{value: pfs.token, errMsg: "missing token"},
	} {
		if field.value == "" {
			return fmt.Errorf(field.errMsg)
		}
	}

	var err error
	pfs.client, err = papi.New(pfs.url)
	if err != nil {
		return err
	}
	pfs.client.SetToken(pfs.token)

	return nil
}

func (pfs *polochonfs) buildFS(ctx context.Context) {
	fmt.Println("Fecthing movies")
	movies, err := pfs.client.GetMovies()
	if err != nil {
		fmt.Println("Failed to get movies: ", err)
		return
	}
	pfs.movies = movies.List()

	fmt.Println("Fecthing shows")
	shows, err := pfs.client.GetShows()
	if err != nil {
		fmt.Println("Failed to get shows: ", err)
		return
	}
	pfs.shows = shows.List()

	fmt.Println("Adding persistent nodes")
	pfs.root.addChild(newPersistentNodeDir("movies"))
	pfs.root.addChild(newPersistentNodeDir("shows"))

	pfs.updateMovies(ctx)
	pfs.updateShows(ctx)

	fmt.Println("All done")
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
