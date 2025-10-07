package main

import (
	"sync"
	"time"
)

type item struct {
	value      bool
	expiration int64 // UnixNano timestamp; 0 means no expiration
}

type Cache struct {
	mu    sync.RWMutex
	items map[string]item
}

func New() *Cache {
	return &Cache{
		items: make(map[string]item),
	}
}

// Set stores a bool value with a TTL. ttl = 0 means no expiration.
func (c *Cache) Set(key string, value bool, ttl time.Duration) {
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).UnixNano()
	}

	c.mu.Lock()
	c.items[key] = item{
		value:      value,
		expiration: exp,
	}
	c.mu.Unlock()
}

// Get retrieves a bool value if it exists and isn’t expired.
func (c *Cache) Get(key string) (bool, bool) {
	c.mu.RLock()
	it, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		return false, false
	}

	if it.expiration > 0 && time.Now().UnixNano() > it.expiration {
		// expired — delete and return not found
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return false, false
	}

	return it.value, true
}
