package main

import (
	"bytes"
	"context"
	"io"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	log "github.com/sirupsen/logrus"
)

// blockSize represents the read size off the HTTP body.
var blockSize int64 = 256_000 // 256KB

// asyncReader is a reader that reads from a source, cache some data and can be
// read from.
type asyncReader struct {
	name   string
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	sourcePos, bufferPos int64
	end                  int64

	mu     sync.RWMutex
	source io.ReadCloser
	buffer bytes.Buffer
}

// cacheSize returns the current cache size.
func (r *asyncReader) cacheSize() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.buffer.Len()
}

// readBlockFromSource reads `blockSize` from the source into the cache.
func (r *asyncReader) readBlockFromSource() error {
	n := min(blockSize, r.end-r.sourcePos)

	r.mu.Lock()
	read, err := io.CopyN(&r.buffer, r.source, n)
	r.mu.Unlock()
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"name":       r.name,
		"bytes":      humanize.SI(float64(n), "B"),
		"offset":     r.sourcePos,
		"cache_size": humanize.SI(float64(r.cacheSize()), "B"),
		"read":       read,
	}).Debug("Async read block from source")

	r.sourcePos += read

	if r.sourcePos == r.end {
		r.source.Close()
	}

	return err
}

// Read implements the io.Reader interface
func (r *asyncReader) Read(p []byte) (int, error) {
	if r.bufferPos == r.end {
		return 0, io.EOF
	}

	requested := min(len(p), int(r.end-r.bufferPos))

	ctx, _ := context.WithTimeout(r.ctx, defaultTimeout)
	try := 0
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		if r.cacheSize() >= requested {
			break
		}

		if try%10 == 0 {
			log.WithFields(log.Fields{
				"name":       r.name,
				"cache_size": humanize.SI(float64(r.cacheSize()), "B"),
				"requested":  requested,
				"tries":      try,
			}).Debug("Async read not enough bytes to read")
		}

		// Wait until the buffer has enough bytes to read.
		time.Sleep(time.Millisecond)
		try++
	}

	r.mu.Lock()
	read, err := r.buffer.Read(p)
	r.mu.Unlock()
	r.bufferPos += int64(read)
	return read, err
}

// Close implements the io.Closer interface
func (r *asyncReader) Close() error {
	r.cancel()
	r.wg.Wait()
	r.source.Close()

	log.WithField("name", r.name).Debug("Async buffer closed")
	return nil
}

// asyncRead reads data until the cache reaches `cacheSize`.
func (r *asyncReader) asyncRead() {
	defer r.wg.Done()
	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		if r.sourcePos == r.end {
			return
		}

		if r.cacheSize() >= cacheSize {
			time.Sleep(time.Millisecond)
			continue
		}

		if err := r.readBlockFromSource(); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"name":  r.name,
			}).Error("Async read error")
			return
		}
	}
}

// newAsyncReader creates a new asyncReader and starts a goroutine to read the
// data asynchronously from the source.
func newAsyncReader(ctx context.Context, name string, s io.ReadCloser, start, end int64) *asyncReader {
	r := &asyncReader{
		name:      name,
		sourcePos: start,
		bufferPos: start,
		end:       end,
		source:    s,
		buffer:    bytes.Buffer{},
	}

	r.ctx, r.cancel = context.WithCancel(ctx)
	r.wg.Add(1)

	go r.asyncRead()

	log.WithFields(log.Fields{
		"name":       r.name,
		"start":      start,
		"end":        end,
		"total_size": humanize.SI(float64(start-end), "B"),
		"block_size": humanize.SI(float64(blockSize), "B"),
		"cache_size": humanize.SI(float64(cacheSize), "B"),
	}).Debug("Async buffer created")
	return r
}
