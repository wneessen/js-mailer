// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package cache

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/wneessen/js-mailer/internal/forms"
)

var testForm = &forms.Form{ID: "test"}

func TestNew(t *testing.T) {
	t.Run("new returns a cache", func(t *testing.T) {
		interval := time.Millisecond * 10
		cache := New(interval)
		if cache == nil {
			t.Fatal("cache is nil")
		}
		t.Cleanup(cache.Stop)
		if cache.ttl != interval {
			t.Errorf("expected ttl to be %s, got %s", interval, cache.ttl)
		}
		if cache.stop == nil {
			t.Error("expected stop to be non-nil")
		}
		if cache.items == nil {
			t.Error("expected items to be non-nil")
		}
	})
}

func TestCache_Set(t *testing.T) {
	t.Run("set adds an item to the cache", func(t *testing.T) {
		interval := time.Millisecond * 10
		key := "test"
		cache := New(interval)
		if cache == nil {
			t.Fatal("cache is nil")
		}

		cache.Set(key, testForm, ItemParams{
			TokenCreatedAt: time.Now(),
			TokenExpiresAt: time.Now().Add(interval),
		})
		if _, ok := cache.items[key]; !ok {
			t.Error("item was not added to the cache")
		}
	})
}

func TestCache_Get(t *testing.T) {
	interval := time.Millisecond * 10
	key := "test"
	now := time.Now()

	t.Run("get returns an item from the cache", func(t *testing.T) {
		expireAt := now.Add(interval)
		cache := New(interval)
		if cache == nil {
			t.Fatal("cache is nil")
		}

		cache.Set(key, testForm, ItemParams{
			TokenCreatedAt: now,
			TokenExpiresAt: expireAt,
		})
		form, params, exists := cache.Get(key)
		if !exists {
			t.Error("item was not found in the cache")
		}
		if form.ID != testForm.ID {
			t.Errorf("expected form to be %s, got %s", testForm.ID, form.ID)
		}
		if params.TokenCreatedAt != now {
			t.Errorf("expected created at to be %s, got %s", now, params.TokenCreatedAt)
		}
		if params.TokenExpiresAt != expireAt {
			t.Errorf("expected expires at to be %s, got %s", expireAt, params.TokenExpiresAt)
		}
	})
	t.Run("get does not find cache item", func(t *testing.T) {
		expireAt := now.Add(interval)
		cache := New(interval)
		if cache == nil {
			t.Fatal("cache is nil")
		}

		cache.Set(key, testForm, ItemParams{TokenCreatedAt: now, TokenExpiresAt: expireAt})
		_, _, exists := cache.Get(key + "non-existing")
		if exists {
			t.Error("item was expected to not exist in the cache")
		}
	})
	t.Run("cache item is expired while trying to get it", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			interval = time.Second
			cache := New(interval)
			// Make sure the auto cleanup goroutine is stopped
			cache.Stop()

			cache.Set("key", testForm, ItemParams{
				TokenCreatedAt: time.Now(),
				TokenExpiresAt: time.Now().Add(interval),
			})
			time.Sleep(interval + 1)
			synctest.Wait()
			_, _, exists := cache.Get("key")
			if exists {
				t.Error("item was expected to be expired")
			}
		})
	})
}

func TestCache_Remove(t *testing.T) {
	t.Run("remove a cached item", func(t *testing.T) {
		interval := time.Millisecond * 100
		key := "test"
		now := time.Now()
		expireAt := now.Add(interval)
		cache := New(interval)
		if cache == nil {
			t.Fatal("cache is nil")
		}

		cache.Set(key, testForm, ItemParams{TokenCreatedAt: now, TokenExpiresAt: expireAt})
		cache.Remove(key)
		_, _, exists := cache.Get(key)
		if exists {
			t.Error("item was expected to be removed from the cache")
		}
	})
}

func TestCache_cleanupLoop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		interval := time.Second
		cache := New(interval)
		cache.Start()
		t.Cleanup(cache.Stop)

		cache.Set("key", testForm, ItemParams{
			TokenCreatedAt: time.Now(),
			TokenExpiresAt: time.Now().Add(interval),
		})
		time.Sleep(interval * 2)
		synctest.Wait()
		_, _, exists := cache.Get("key")
		if exists {
			t.Error("item was expected to be expired")
		}
	})
}
