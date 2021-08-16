package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/odwrtw/polochon/lib/papi"
)

type fs struct {
	mountPoint string
	root       *node
	client     *papi.Client

	debug    bool
	mutex    sync.Mutex
	nextFree uint64
}

func newFS(mountPoint, url, token string, debug bool) (*fs, error) {
	fs := &fs{
		mountPoint: mountPoint,
		debug:      debug,
	}

	client, err := papi.New(url)
	if err != nil {
		return nil, err
	}
	client.SetToken(token)
	fs.client = client

	fs.root = fs.newNode()
	now := time.Now()
	fs.root.info.SetTimes(&now, &now, &now)
	fs.root.info.Mode = fuse.S_IFDIR | 0777

	return fs, nil
}

func (fs *fs) newNode() *node {
	fs.mutex.Lock()
	id := fs.nextFree
	fs.nextFree++
	fs.mutex.Unlock()

	return &node{
		Node: nodefs.NewDefaultNode(),
		fs:   fs,
		id:   id,
	}
}

func (fs *fs) stats() *fuse.StatfsOut {
	var inodesMax uint64 = ^uint64(0)
	return &fuse.StatfsOut{
		Files:   inodesMax,
		Ffree:   inodesMax - fs.nextFree,
		Bsize:   4096,
		NameLen: 255,
	}
}

func (fs *fs) mount() (*fuse.Server, error) {
	options := nodefs.NewOptions()
	options.Debug = fs.debug

	conn := nodefs.NewFileSystemConnector(fs.root, options)

	mountOpts := fuse.MountOptions{
		// Mount as read only for now
		Options: []string{"ro"},
		Name:    "polochonfs",
		FsName:  "polochonfs",
		Debug:   fs.debug,
		// Disable extended attributes
		DisableXAttrs: true,
		// FUSE will wait for one callback to return before calling another
		SingleThreaded: true,
	}

	return fuse.NewServer(conn.RawFS(), fs.mountPoint, &mountOpts)
}

type node struct {
	nodefs.Node

	id  uint64
	fs  *fs
	url string

	mu   sync.Mutex
	info fuse.Attr
}

func (n *node) GetAttr(out *fuse.Attr, file nodefs.File, context *fuse.Context) fuse.Status {
	n.mu.Lock()
	defer n.mu.Unlock()

	*out = n.info
	return fuse.OK
}

func (n *node) StatFs() *fuse.StatfsOut {
	return n.fs.stats()
}

func (n *node) addDir(name string, t time.Time) *node {
	newNode := n.fs.newNode()
	newNode.info.Mode = fuse.S_IFDIR | 0770
	newNode.info.SetTimes(&t, &t, &t)
	n.Inode().NewChild(name, true, newNode)
	return newNode
}

func (n *node) add(name, url string, t time.Time, size int64) *node {
	newNode := n.fs.newNode()
	newNode.url = url
	newNode.info.Mode = fuse.S_IFREG | 0440
	newNode.info.SetTimes(&t, &t, &t)
	newNode.info.Size = uint64(size)
	n.Inode().NewChild(name, false, newNode)
	return newNode
}

func (n *node) Open(flags uint32, context *fuse.Context) (nodefs.File, fuse.Status) {
	file := newFile()
	file.node = n
	return file, fuse.OK
}

type file struct {
	nodefs.File

	data       io.ReadCloser
	lastOffset int64

	node *node
}

func newFile() *file {
	return &file{
		File: nodefs.NewDefaultFile(),
	}
}

func (f *file) setupData(off int64) error {
	if f.data != nil && f.lastOffset == off {
		log.Printf("No need to redo a http call")
		return nil
	}

	if f.data != nil {
		log.Printf("Closing the old data")
		if err := f.data.Close(); err != nil {
			return err
		}
	}

	log.Printf("Making a http call")
	client := &http.Client{}

	req, err := http.NewRequest("GET", f.node.url, nil)
	if err != nil {
		return err
	}

	if off != 0 {
		req.Header.Add("Range", fmt.Sprintf("bytes=%d-", off))
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("got HTTP response: %s", resp.Status)
	}

	f.data = resp.Body
	f.lastOffset = off

	return nil
}

func (f *file) Read(dest []byte, off int64) (fuse.ReadResult, fuse.Status) {
	if err := f.setupData(off); err != nil {
		log.Printf("failed to setup data: %s", err.Error())
		return nil, fuse.EAGAIN
	}

	var n, total int
	var err error

	for {
		n, err = f.data.Read(dest[total:])
		total += n

		if n == 0 || err != nil {
			break
		}
	}

	if err != nil && err != io.EOF {
		return nil, fuse.ToStatus(err)
	}

	f.lastOffset += int64(total)

	return fuse.ReadResultData(dest), fuse.OK
}

func (f *file) Flush() fuse.Status {
	if f.data == nil {
		return fuse.OK
	}

	if err := f.data.Close(); err != nil {
		return fuse.ToStatus(err)
	}

	return fuse.OK
}
