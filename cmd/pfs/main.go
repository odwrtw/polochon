package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

const (
	polochonEndpoint = "http://localhost:8080"
	polochonToken    = "toto"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()
	if len(flag.Args()) < 1 {
		return fmt.Errorf("missing mountpoint")
	}
	mountPoint := flag.Arg(0)

	fs, err := newFS(mountPoint, polochonEndpoint, polochonToken, true)
	if err != nil {
		return err
	}

	server, err := fs.mount()
	if err != nil {
		return err
	}

	if err := fs.handleMovies(); err != nil {
		return err
	}

	if err := fs.handleShows(); err != nil {
		return err
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		server.Unmount()
	}()

	server.Serve()
	return nil
}
