// Package cache is a small in-memory cache with a time to live.
package cache

import (
	"sync"
	"time"
)

type entry[V any] struct {
	value   V
	expires time.Time
}

// TTL is safe for concurrent use. Expired entries are removed when read
// or overwritten, so memory is bounded by the number of distinct keys.
type TTL[K comparable, V any] struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[K]entry[V]
	now     func() time.Time // replaced in tests
}

func NewTTL[K comparable, V any](ttl time.Duration) *TTL[K, V] {
	return &TTL[K, V]{ttl: ttl, entries: map[K]entry[V]{}, now: time.Now}
}

func (c *TTL[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if ok && c.now().Before(e.expires) {
		return e.value, true
	}
	if ok {
		delete(c.entries, key)
	}
	var zero V
	return zero, false
}

func (c *TTL[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = entry[V]{value: value, expires: c.now().Add(c.ttl)}
}

func (c *TTL[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

func (c *TTL[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	clear(c.entries)
}
