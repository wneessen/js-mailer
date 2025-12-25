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
	params     ItemParams
	expiration time.Time
}

type ItemParams struct {
	TokenCreatedAt   time.Time
	TokenExpiresAt   time.Time
	RandomFieldName  string
	RandomFieldValue string
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
	return c
}

func (c *Cache) Start() {
	go c.cleanupLoop(c.ttl)
}

// Set stores a value without TTL.
func (c *Cache) Set(key string, form *forms.Form, params ItemParams) {
	c.mu.Lock()
	c.items[key] = &item{
		form:       form,
		params:     params,
		expiration: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

// Get retrieves a value. Second return value indicates presence.
func (c *Cache) Get(key string) (*forms.Form, ItemParams, bool) {
	c.mu.RLock()
	cacheItem, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		return nil, ItemParams{}, false
	}

	if !cacheItem.expiration.IsZero() && time.Now().After(cacheItem.expiration) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, ItemParams{}, false
	}

	return cacheItem.form, cacheItem.params, true
}

func (c *Cache) Remove(key string) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
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
