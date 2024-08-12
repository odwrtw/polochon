package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type node struct {
	fs.Inode
	isDir, isPersistent bool
	url                 string

	times time.Time
	name  string
	size  uint64

	mu     sync.Mutex
	childs map[string]*node
}

func newNodeDir(name string) *node {
	return &node{
		name:   name,
		isDir:  true,
		childs: map[string]*node{},
	}
}

func newPersistentNodeDir(name string) *node {
	n := newNodeDir(name)
	n.isPersistent = true
	return n
}

func newRootNode() *node {
	n := newNodeDir("root")
	n.times = time.Now()
	return n
}

func newNode(name, url string, size uint64, times time.Time) *node {
	return &node{
		isDir:  false,
		name:   name,
		url:    url,
		size:   size,
		times:  times,
		childs: map[string]*node{},
	}
}

func (n *node) addChildNode(c *node) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.childs[c.name] = c
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
	defer n.mu.Unlock()

	entries := make([]fuse.DirEntry, 0, len(n.childs))
	for _, c := range n.childs {
		entry := fuse.DirEntry{
			Mode: c.Mode(),
			Name: c.name,
			Ino:  c.StableAttr().Ino,
		}

		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	return fs.NewListDirStream(entries), 0
}

func (n *node) updateAttr(out *fuse.Attr) {
	out.Uid = uid
	out.Gid = gid
	out.SetTimes(&n.times, &n.times, &n.times)

	if n.isDir {
		out.Size = 4096
	} else {
		out.Size = n.size
	}
}

func (n *node) Getattr(ctx context.Context, _ fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	fmt.Println("Getattr on file:", n.name)
	n.updateAttr(&out.Attr)
	return 0
}

func (n *node) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	fmt.Printf("Looking up %q on node %q\n", name, n.name)
	child := n.getChild(name)
	if child == nil {
		return nil, syscall.ENOENT
	}

	child.updateAttr(&out.Attr)
	return &child.Inode, 0
}

func (n *node) Open(ctx context.Context, flags uint32) (fh fs.FileHandle, fuseFlags uint32, errno syscall.Errno) {
	fmt.Printf("Open called on %s with flags %d\n", n.name, flags)
	return n, 0, 0
}

func (n *node) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	fmt.Printf("Reading node %s at offset %d\n", n.name, off)
	client := &http.Client{Timeout: httpTimeout}
	defaultErr := syscall.ENETUNREACH // Network unreachable

	fmt.Printf("Fetching URL: %s\n", n.url)
	req, err := http.NewRequestWithContext(ctx, "GET", n.url, nil)
	if err != nil {
		return fuse.ReadResultData(dest), defaultErr
	}
	req.Header.Add("X-Auth-Token", polochonToken)

	if off != 0 {
		req.Header.Add("Range", fmt.Sprintf("bytes=%d-", off))
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Failed to do http request: ", err)
		return fuse.ReadResultData(dest), defaultErr
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		fmt.Println("Invalid HTTP response code: ", resp.Status)
		return fuse.ReadResultData(dest), defaultErr
	}

	_, err = io.ReadFull(resp.Body, dest)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		fmt.Println("Failed to read response body: ", err)
		return fuse.ReadResultData(dest), defaultErr
	}

	return fuse.ReadResultData(dest), 0
}
