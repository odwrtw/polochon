package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	log "github.com/sirupsen/logrus"
)

var (
	_ = (fs.NodeGetattrer)((*node)(nil))
	_ = (fs.NodeOpener)((*node)(nil))

	_ = (fs.FileReader)((*fileHandle)(nil))
	_ = (fs.FileFlusher)((*fileHandle)(nil))
)

type node struct {
	fs.Inode

	times time.Time
	name  string
	size  uint64
	url   string
	isDir bool

	mu       sync.Mutex
	children map[string]*node
	valid    bool
}

func newNode(name string, isDir bool) *node {
	n := &node{
		name:     name,
		children: map[string]*node{},
		isDir:    isDir,
	}

	return n
}

func newNodeDir(name string, times time.Time) *node {
	node := newNode(name, true)
	node.times = times
	return node
}

func newRootNode() *node {
	return &node{
		name:     "root",
		times:    time.Now(),
		children: map[string]*node{},
	}
}

func newNodeFile(name string) *node {
	return newNode(name, false)
}

func (n *node) setURL(url string) {
	n.url = url
}

func (n *node) invalidate() {
	n.mu.Lock()
	defer n.mu.Unlock()

	for _, child := range n.children {
		child.invalidate()
	}

	n.valid = false
}

func (n *node) clear() {
	n.mu.Lock()
	defer n.mu.Unlock()

	for name, child := range n.children {
		child.clear()
		if child.valid {
			continue
		}

		log.WithFields(log.Fields{
			"parent": n.name,
			"child":  name,
		}).Debug("Removing child")

		ok, _ := n.Inode.RmChild(name)
		if !ok {
			log.WithFields(log.Fields{
				"parent": n.name,
				"child":  name,
				"ok":     ok,
			}).Error("Failed to remove inode child")
			continue
		}

		if ret := n.NotifyDelete(name, &child.Inode); ret != 0 {
			log.WithField("file", child.name).Error("Failed to notify delete")
		}

		delete(n.children, name)
	}
}

func (n *node) addChildNode(c *node) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.children[c.name] = c
}

func (n *node) addChild(child *node) {
	attr := fs.StableAttr{Mode: syscall.S_IFREG}
	if child.isDir {
		attr.Mode = syscall.S_IFDIR
	}

	inode := n.NewInode(context.Background(), child, attr)
	n.addChildNode(child)
	n.AddChild(child.name, inode, true)
}

func (n *node) getChild(name string) *node {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.children[name]
}

func (n *node) childCount() uint32 {
	n.mu.Lock()
	defer n.mu.Unlock()
	return uint32(len(n.children))
}

func (n *node) updateAttr(out *fuse.Attr) {
	out.SetTimes(&n.times, &n.times, &n.times)

	if n.IsDir() {
		out.Size = 4096
		out.Nlink = n.childCount()
	} else {
		out.Size = n.size
		out.Blksize = 4096 * 4
	}
}

func (n *node) Getattr(_ context.Context, _ fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	log.WithField("node", n.name).Debug("Getattr called")
	out.SetTimeout(libraryRefresh)
	n.updateAttr(&out.Attr)
	return 0
}

func (n *node) Open(_ context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	log.WithFields(log.Fields{
		"node":  n.name,
		"flags": flags,
	}).Debug("Open called on node")
	return newFileHandle(n.name, n.url, int64(n.size)), 0, 0
}

type fileHandle struct {
	name, url        string
	size, lastOffset int64
	cancel           context.CancelFunc
	buffer           io.ReadCloser
}

func newFileHandle(name, url string, size int64) *fileHandle {
	return &fileHandle{
		name: name,
		url:  url,
		size: size,
	}
}

func (fh *fileHandle) Flush(_ context.Context) syscall.Errno {
	log.WithField("name", fh.name).Debug("Flush called")
	fh.close()
	return 0
}

func (fh *fileHandle) close() {
	if fh.buffer == nil {
		return
	}
	fh.cancel()
	_ = fh.buffer.Close()
	fh.buffer = nil
}

func (fh *fileHandle) setup(_ context.Context, offset int64) error {
	log.WithFields(log.Fields{
		"name":   fh.name,
		"offset": offset,
	}).Trace("Setting up filehandle")

	cancelCtx, cancelFunc := context.WithCancel(globalCtx)
	fh.cancel = cancelFunc

	client := http.DefaultClient
	req, err := http.NewRequestWithContext(cancelCtx, "GET", fh.url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("X-Auth-Token", polochonToken)
	req.Header.Add("User-Agent", "polochonfs")

	if offset != 0 {
		req.Header.Add("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	timeout := time.AfterFunc(defaultTimeout, func() {
		log.WithField("name", fh.name).Error("Request timeout")
		fh.cancel()
	})

	resp, err := client.Do(req)
	timeout.Stop()
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("Invalid HTTP response code")
	}

	if fh.buffer != nil {
		log.WithField("name", fh.name).Trace("Closing old buffer")
		_ = fh.buffer.Close()
		log.WithField("name", fh.name).Trace("Done closing old buffer")
	}

	fh.lastOffset = offset
	fh.buffer = newAsyncReader(cancelCtx, fh.name, resp.Body, offset, fh.size)
	return nil
}

func (fh *fileHandle) Read(ctx context.Context, dest []byte, offset int64) (fuse.ReadResult, syscall.Errno) {
	readSize := int64(len(dest))

	l := log.WithFields(log.Fields{
		"name":             fh.name,
		"requested_offset": offset,
	})

	// Handle context cancelled from the given fuse.Context
	err := ctx.Err()
	if err != nil {
		if err == context.Canceled {
			// go-fuse advises to return EINTR on canceled context.
			return fuse.ReadResultData(dest), syscall.EINTR
		}

		l.WithField("error", err).Error("Read failed from fuse context")
		return fuse.ReadResultData(dest), syscall.EIO
	}

	// Handle the global context
	err = globalCtx.Err()
	if err != nil {
		if err != context.Canceled {
			l.WithField("error", err).Error("Read failed from global context")
		}
		return fuse.ReadResultData(dest), syscall.EIO
	}

	defaultErr := syscall.ENETUNREACH // Network unreachable
	if fh.buffer == nil || offset != fh.lastOffset {
		if err := fh.setup(ctx, offset); err != nil {
			l.WithField("error", err).Error("Failed to setup file handle")
			return fuse.ReadResultData(dest), defaultErr
		}
	}

	timedOut := false
	timeout := time.AfterFunc(defaultTimeout, func() {
		timedOut = true
		fh.cancel()
	})
	read, err := fh.buffer.Read(dest)
	timeout.Stop()

	fh.lastOffset += int64(read)
	l = l.WithFields(log.Fields{
		"read":        humanize.SI(float64(readSize), "B"),
		"last_offset": fh.lastOffset,
	})

	if timedOut {
		err = fmt.Errorf("timeout after %s", defaultTimeout.String())
	}

	switch err {
	case nil:
		l.Trace("Read from async reader")
		return fuse.ReadResultData(dest), 0
	case io.EOF:
		l.Trace("Read from async reader until EOF")
		return fuse.ReadResultData(dest), 0
	case context.Canceled:
		l.Debug("Context cancelled")
	default:
		l.WithField("error", err).Error("Failed to read from async reader")
	}

	fh.close()
	return fuse.ReadResultData(dest), defaultErr
}
