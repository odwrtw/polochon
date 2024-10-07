package main

import (
	"bytes"
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"strconv"
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
	isDir bool
	url   string

	times    time.Time
	id, name string
	size     uint64

	mu       sync.Mutex
	children map[string]*node

	inode uint64
}

func newNodeDir(id, name string, times time.Time) *node {
	return &node{
		id:       id,
		name:     name,
		isDir:    true,
		times:    times,
		children: map[string]*node{},
	}
}

func newRootNode() *node {
	return newNodeDir("root", "root", time.Now())
}

func newNode(id, name, url string, size uint64, times time.Time) *node {
	return &node{
		id:       id,
		isDir:    false,
		name:     name,
		url:      url,
		size:     size,
		times:    times,
		children: map[string]*node{},
	}
}

func (n *node) getInode() uint64 {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.inode != 0 {
		return n.inode
	}

	var b bytes.Buffer
	if !n.isDir {
		b.WriteString(n.times.String())
	}

	b.WriteString(n.id)
	b.WriteString(strconv.FormatUint(n.size, 10))
	b.WriteString(n.name)

	n.inode = baseInode | uint64(crc32.ChecksumIEEE(b.Bytes()))

	return n.inode
}

func (n *node) addChildNode(c *node) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.children[c.name] = c
}

func inodeExists(i uint64) bool {
	if _, ok := movieInodes[i]; ok {
		return true
	}
	if _, ok := showInodes[i]; ok {
		return true
	}
	return false
}

func (n *node) addChild(child *node) {
	attr := fs.StableAttr{
		Ino: child.getInode(),
	}
	if child.isDir {
		attr.Mode = syscall.S_IFDIR
	}

	// Check for inode collision.
	if inodeExists(attr.Ino) {
		log.WithFields(log.Fields{
			"file":  n.name,
			"inode": attr.Ino,
		}).Error("Inode already exists")
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

func (n *node) rmAllChildren() {
	n.mu.Lock()
	n.children = map[string]*node{}
	n.mu.Unlock()
	n.RmAllChildren()
}

func (n *node) childCount() uint32 {
	n.mu.Lock()
	defer n.mu.Unlock()
	return uint32(len(n.children))
}

func (n *node) updateAttr(out *fuse.Attr) {
	out.SetTimes(&n.times, &n.times, &n.times)

	if n.isDir {
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
