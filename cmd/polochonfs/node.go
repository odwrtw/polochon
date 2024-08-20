package main

import (
	"bytes"
	"context"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"sort"
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
	_ = (fs.NodeOpener)((*node)(nil))
	_ = (fs.NodeLookuper)((*node)(nil))

	_ = (fs.FileReader)((*fileHandle)(nil))
	_ = (fs.FileFlusher)((*fileHandle)(nil))
)

type node struct {
	fs.Inode
	isDir, isPersistent bool
	url                 string

	times    time.Time
	id, name string
	size     uint64

	mu     sync.Mutex
	childs map[string]*node

	inode uint64
}

func newNodeDir(id, name string) *node {
	return &node{
		id:     id,
		name:   name,
		isDir:  true,
		childs: map[string]*node{},
	}
}

func newPersistentNodeDir(name string) *node {
	n := newNodeDir(name, name)
	n.isPersistent = true
	return n
}

func newRootNode() *node {
	n := newNodeDir("root", "root")
	n.times = time.Now()
	return n
}

func newNode(id, name, url string, size uint64, times time.Time) *node {
	return &node{
		isDir:  false,
		name:   name,
		url:    url,
		size:   size,
		times:  times,
		childs: map[string]*node{},
	}
}

func (n *node) getInode() uint64 {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.inode != 0 {
		return n.inode
	}

	var b bytes.Buffer
	if !n.isPersistent {
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
	n.childs[c.name] = c
}

func checkInodeExist(i uint64) bool {
	if _, ok := movieInodes[i]; ok {
		return true
	}
	if _, ok := showInodes[i]; ok {
		return true
	}
	return false
}

func (n *node) addChild(child *node) {
	attr := fs.StableAttr{}
	if child.isDir {
		attr.Mode = syscall.S_IFDIR
	}

	if child.isPersistent {
		child.times = n.times
		n.NewPersistentInode(context.Background(), child, attr)
	} else {
		attr.Ino = child.getInode()
		if checkInodeExist(attr.Ino) {
			log.WithFields(log.Fields{
				"file":  n.name,
				"inode": attr.Ino,
			}).Error("Inode already exists")
		}
		n.NewInode(context.Background(), child, attr)
	}

	n.addChildNode(child)
}

func (n *node) getChild(name string) *node {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.childs[name]
}

func (n *node) rmAllChilds() {
	n.mu.Lock()
	n.childs = map[string]*node{}
	n.mu.Unlock()
	n.RmAllChildren()
}

func (n *node) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	if !n.isDir {
		return nil, syscall.ENOTDIR
	}

	n.mu.Lock()
	entries := make([]fuse.DirEntry, 0, len(n.childs))
	for _, c := range n.childs {
		entry := fuse.DirEntry{
			Mode: c.Mode(),
			Name: c.name,
			Ino:  c.StableAttr().Ino,
		}

		entries = append(entries, entry)
	}
	n.mu.Unlock()

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return fs.NewListDirStream(entries), 0
}

func (n *node) childCount() uint32 {
	n.mu.Lock()
	defer n.mu.Unlock()
	return uint32(len(n.childs))
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

func (n *node) Getattr(ctx context.Context, _ fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	log.WithField("node", n.name).Debug("Getattr called")
	out.SetTimeout(libraryRefresh)
	n.updateAttr(&out.Attr)
	return 0
}

func (n *node) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	log.WithFields(log.Fields{
		"node":   n.name,
		"lookup": name,
	}).Debug("Looking up node")

	child := n.getChild(name)
	if child == nil {
		return nil, syscall.ENOENT
	}

	child.updateAttr(&out.Attr)
	return &child.Inode, 0
}

func (n *node) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	log.WithFields(log.Fields{
		"node":  n.name,
		"flags": flags,
	}).Debug("Open called on node")
	return newFileHandle(n.name, n.url, int64(n.size)), 0, 0
}

type fileHandle struct {
	name, url        string
	size, lastOffset int64
	buffer           io.ReadCloser
}

func newFileHandle(name, url string, size int64) *fileHandle {
	return &fileHandle{
		name: name,
		url:  url,
		size: size,
	}
}

func (fh *fileHandle) Flush(ctx context.Context) syscall.Errno {
	log.WithField("name", fh.name).Debug("Flush called")
	fh.close()
	return 0
}

func (fh *fileHandle) close() {
	if fh.buffer == nil {
		return
	}
	fh.buffer.Close()
	fh.buffer = nil
}

func (fh *fileHandle) setup(ctx context.Context, offset int64) error {
	log.WithFields(log.Fields{
		"name":   fh.name,
		"offset": offset,
	}).Debug("Setting up filehandle")

	client := http.DefaultClient
	req, err := http.NewRequestWithContext(ctx, "GET", fh.url, nil)
	if err != nil {
		return err
	}
	req.Header.Add("X-Auth-Token", polochonToken)
	req.Header.Add("User-Agent", "polochonfs")

	if offset != 0 {
		req.Header.Add("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("Invalid HTTP response code")
	}

	if fh.buffer != nil {
		log.WithField("name", fh.name).Debug("Closing old buffer")
		fh.buffer.Close()
		log.WithField("name", fh.name).Debug("Done closing old buffer")
	}

	fh.lastOffset = offset
	fh.buffer = newAsyncReader(ctx, fh.name, resp.Body, offset, fh.size)
	return nil
}

func (fh *fileHandle) Read(ctx context.Context, dest []byte, offset int64) (fuse.ReadResult, syscall.Errno) {
	defaultErr := syscall.ENETUNREACH // Network unreachable
	readSize := int64(len(dest))

	l := log.WithFields(log.Fields{
		"name":             fh.name,
		"requested_offset": offset,
	})

	err := ctx.Err()
	if err != nil {
		l.WithField("error", err).Error("Read failed")
		return fuse.ReadResultData(dest), defaultErr
	}

	if fh.buffer == nil || offset != fh.lastOffset {
		if err := fh.setup(ctx, offset); err != nil {
			l.WithField("error", err).Error("Failed to setup file handle")
			return fuse.ReadResultData(dest), defaultErr
		}
	}

	read, err := fh.buffer.Read(dest)
	fh.lastOffset += int64(read)
	l = l.WithFields(log.Fields{
		"read":        humanize.SI(float64(readSize), "B"),
		"last_offset": fh.lastOffset,
	})
	switch err {
	case nil:
		l.Debug("Read from async reader")
		return fuse.ReadResultData(dest), 0
	case io.EOF:
		l.Debug("Read from async reader until EOF")
		return fuse.ReadResultData(dest), 0
	case context.Canceled:
		l.Debug("Context cancelled")
	default:
		l.WithField("error", err).Error("Failed to read from async reader")
	}

	fh.close()
	return fuse.ReadResultData(dest), defaultErr
}
