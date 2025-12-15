// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package cache

import (
	"sync"
	"time"

	"github.com/wneessen/js-mailer/internal/forms"
)

type item struct {
	form       *forms.Form
	expiration time.Time
}

type Cache struct {
	mu    sync.RWMutex
	items map[string]*item
	ttl   time.Duration
	stop  chan struct{}
}

// New creates a cache and starts the cleanup goroutine.
func New(cleanupInterval time.Duration) *Cache {
	c := &Cache{
		items: make(map[string]*item),
		ttl:   cleanupInterval,
		stop:  make(chan struct{}),
	}
	go c.cleanupLoop(cleanupInterval)
	return c
}

// Set stores a value without TTL.
func (c *Cache) Set(key string, form *forms.Form) {
	c.mu.Lock()
	c.items[key] = &item{form: form, expiration: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

// Get retrieves a value. Second return value indicates presence.
func (c *Cache) Get(key string) (*forms.Form, bool) {
	c.mu.RLock()
	cacheItem, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if !cacheItem.expiration.IsZero() && time.Now().After(cacheItem.expiration) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}

	return cacheItem.form, true
}

// Stop shuts down the cleanup goroutine.
func (c *Cache) Stop() {
	close(c.stop)
}

func (c *Cache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			c.mu.Lock()
			for k, it := range c.items {
				if !it.expiration.IsZero() && now.After(it.expiration) {
					delete(c.items, k)
				}
			}
			c.mu.Unlock()

		case <-c.stop:
			return
		}
	}
}
