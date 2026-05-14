package cache

import (
	"sync"
	"time"
)

// Cache is a thread-safe in-memory cache with TTL support
type Cache struct {
	data map[string]entry
	mu   sync.RWMutex
	ttl  time.Duration
}

type entry struct {
	value     any
	expiresAt time.Time
}

// New creates a new cache with the specified TTL
func New(ttl time.Duration) *Cache {
	c := &Cache{
		data: make(map[string]entry),
		ttl:  ttl,
	}
	// Start background cleanup goroutine
	go c.cleanup()
	return c
}

// Get retrieves a value from the cache
// Returns the value and true if found and not expired, otherwise nil and false
func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	e, ok := c.data[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(e.expiresAt) {
		return nil, false
	}

	return e.value, true
}

// Set stores a value in the cache with the default TTL
func (c *Cache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = entry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// SetWithTTL stores a value in the cache with a custom TTL
func (c *Cache) SetWithTTL(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = entry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, key)
}

// Clear removes all entries from the cache
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]entry)
}

// Size returns the number of entries in the cache (including expired ones)
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.data)
}

// cleanup periodically removes expired entries
func (c *Cache) cleanup() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()

	for range ticker.C {
		c.removeExpired()
	}
}

// removeExpired removes all expired entries from the cache
func (c *Cache) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, e := range c.data {
		if now.After(e.expiresAt) {
			delete(c.data, key)
		}
	}
}
