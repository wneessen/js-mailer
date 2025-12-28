// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package inmemory

import (
	"errors"
	"sync"
	"time"

	"github.com/wneessen/js-mailer/internal/cache"
	"github.com/wneessen/js-mailer/internal/forms"
)

type item struct {
	form       *forms.Form
	params     cache.ItemParams
	expiration time.Time
}

type InMemory struct {
	mu    sync.RWMutex
	items map[string]*item
	ttl   time.Duration
	stop  chan struct{}
}

var (
	ErrItemNotFound = errors.New("cache item not found")
	ErrItemExpired  = errors.New("cache item expired")
)

// New creates a cache and starts the cleanup goroutine.
func New(cleanupInterval time.Duration) *InMemory {
	return &InMemory{
		items: make(map[string]*item),
		ttl:   cleanupInterval,
		stop:  make(chan struct{}),
	}
}

func (i *InMemory) Start() {
	go i.cleanupLoop(i.ttl)
}

// Set stores a value without TTL.
func (i *InMemory) Set(key string, form *forms.Form, params cache.ItemParams) error {
	i.mu.Lock()
	i.items[key] = &item{
		form:       form,
		params:     params,
		expiration: time.Now().Add(i.ttl),
	}
	i.mu.Unlock()
	return nil
}

// Get retrieves a value. Second return value indicates presence.
func (i *InMemory) Get(key string) (*forms.Form, cache.ItemParams, error) {
	i.mu.RLock()
	cacheItem, ok := i.items[key]
	i.mu.RUnlock()

	if !ok {
		return nil, cache.ItemParams{}, ErrItemNotFound
	}

	if !cacheItem.expiration.IsZero() && time.Now().After(cacheItem.expiration) {
		i.mu.Lock()
		delete(i.items, key)
		i.mu.Unlock()
		return nil, cache.ItemParams{}, ErrItemExpired
	}

	return cacheItem.form, cacheItem.params, nil
}

func (i *InMemory) Remove(key string) error {
	i.mu.Lock()
	delete(i.items, key)
	i.mu.Unlock()
	return nil
}

// Stop shuts down the cleanup goroutine.
func (i *InMemory) Stop() {
	close(i.stop)
}

func (i *InMemory) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			i.mu.Lock()
			for k, it := range i.items {
				if !it.expiration.IsZero() && now.After(it.expiration) {
					delete(i.items, k)
				}
			}
			i.mu.Unlock()

		case <-i.stop:
			return
		}
	}
}
