package server

import (
	"sync"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
)

type subtitleCacheEntry struct {
	entries   []*polochon.SubtitleEntry
	expiresAt time.Time
}

type subtitleCache struct {
	mu      sync.Mutex
	entries map[string]*subtitleCacheEntry
	ttl     time.Duration
}

func newSubtitleCache(ttl time.Duration) *subtitleCache {
	return &subtitleCache{
		entries: make(map[string]*subtitleCacheEntry),
		ttl:     ttl,
	}
}

func (c *subtitleCache) get(key string) []*polochon.SubtitleEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, e := range c.entries {
		if now.After(e.expiresAt) {
			delete(c.entries, k)
		}
	}
	e, ok := c.entries[key]
	if !ok {
		return nil
	}
	return e.entries
}

func (c *subtitleCache) set(key string, entries []*polochon.SubtitleEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = &subtitleCacheEntry{
		entries:   entries,
		expiresAt: time.Now().Add(c.ttl),
	}
}
