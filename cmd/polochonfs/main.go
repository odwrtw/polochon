package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"syscall"
	"time"
)

var umountLogTimeout = 1 * time.Minute

var uid, gid uint32

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
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
	flag.StringVar(&pfs.url, "url", os.Getenv("POLOCHON_URL"), "polochon API URL")
	flag.StringVar(&pfs.token, "token", os.Getenv("POLOCHON_TOKEN"), "polochon API token")
	flag.Uint64Var(&uid64, "uid", uid64, "UID of the mounted files")
	flag.Uint64Var(&gid64, "gid", uid64, "GID of the mounted files")
	flag.BoolVar(&pfs.debug, "debug", false, "debug")
	flag.Parse()

	uid = uint32(uid64)
	gid = uint32(gid64)

	err = pfs.init()
	if err != nil {
		flag.PrintDefaults()
		return err
	}

	fmt.Printf("Mouting polochon fs in %s\n", pfs.mountPoint)
	server, err := pfs.mount()
	if err != nil {
		return err
	}

	fmt.Println("FUSE daemon started")
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	cancel()

	pfs.unmount(server)
	return nil
}
