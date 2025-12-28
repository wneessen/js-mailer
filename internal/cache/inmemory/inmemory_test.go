// SPDX-FileCopyrightText: Winni Neessen <wn@neessen.dev>
//
// SPDX-License-Identifier: MIT

package inmemory

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/wneessen/js-mailer/internal/cache"
	"github.com/wneessen/js-mailer/internal/forms"
)

var testForm = &forms.Form{ID: "test"}

func TestNew(t *testing.T) {
	t.Run("new returns a in-memory cache", func(t *testing.T) {
		interval := time.Millisecond * 10
		inmem := New(interval)
		if inmem == nil {
			t.Fatal("in-memory cache is nil")
		}
		t.Cleanup(inmem.Stop)
		if inmem.ttl != interval {
			t.Errorf("expected ttl to be %s, got %s", interval, inmem.ttl)
		}
		if inmem.stop == nil {
			t.Error("expected stop to be non-nil")
		}
		if inmem.items == nil {
			t.Error("expected items to be non-nil")
		}
	})
}

func TestCache_Set(t *testing.T) {
	t.Run("set adds an item to the in-memory cache", func(t *testing.T) {
		interval := time.Millisecond * 10
		key := "test"
		inmem := New(interval)
		if inmem == nil {
			t.Fatal("in-memory cache is nil")
		}

		inmem.Set(key, testForm, cache.ItemParams{
			TokenCreatedAt: time.Now(),
			TokenExpiresAt: time.Now().Add(interval),
		})
		if _, ok := inmem.items[key]; !ok {
			t.Error("item was not added to the in-memory cache")
		}
	})
}

func TestCache_Get(t *testing.T) {
	interval := time.Millisecond * 10
	key := "test"
	now := time.Now()

	t.Run("get returns an item from the in-memory cache", func(t *testing.T) {
		expireAt := now.Add(interval)
		inmem := New(interval)
		if inmem == nil {
			t.Fatal("in-memory cache is nil")
		}

		inmem.Set(key, testForm, cache.ItemParams{
			TokenCreatedAt: now,
			TokenExpiresAt: expireAt,
		})
		form, params, err := inmem.Get(key)
		if err != nil {
			t.Errorf("failed to get item from in-memory cache: %s", err)
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
	t.Run("get does not find in-memory cache item", func(t *testing.T) {
		expireAt := now.Add(interval)
		inmem := New(interval)
		if inmem == nil {
			t.Fatal("in-memory cache is nil")
		}

		inmem.Set(key, testForm, cache.ItemParams{TokenCreatedAt: now, TokenExpiresAt: expireAt})
		_, _, err := inmem.Get(key + "non-existing")
		if err == nil {
			t.Error("item was expected to not exist in the in-memory cache")
		}
	})
	t.Run("in-memory cache item is expired while trying to get it", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			interval = time.Second
			inmem := New(interval)
			// Make sure the auto cleanup goroutine is stopped
			inmem.Stop()

			inmem.Set("key", testForm, cache.ItemParams{
				TokenCreatedAt: time.Now(),
				TokenExpiresAt: time.Now().Add(interval),
			})
			time.Sleep(interval + 1)
			synctest.Wait()
			_, _, err := inmem.Get("key")
			if err == nil {
				t.Error("item was expected to be expired")
			}
		})
	})
}

func TestCache_Remove(t *testing.T) {
	t.Run("remove a in-memory cache item", func(t *testing.T) {
		interval := time.Millisecond * 100
		key := "test"
		now := time.Now()
		expireAt := now.Add(interval)
		inmem := New(interval)
		if inmem == nil {
			t.Fatal("in-memory cache is nil")
		}

		inmem.Set(key, testForm, cache.ItemParams{TokenCreatedAt: now, TokenExpiresAt: expireAt})
		if err := inmem.Remove(key); err != nil {
			t.Errorf("failed to remove item from in-memory cache: %s", err)
		}
		_, _, err := inmem.Get(key)
		if err == nil {
			t.Error("item was expected to be removed from the in-memory cache")
		}
	})
}

func TestCache_cleanupLoop(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		interval := time.Second
		inmem := New(interval)
		inmem.Start()
		t.Cleanup(inmem.Stop)

		inmem.Set("key", testForm, cache.ItemParams{
			TokenCreatedAt: time.Now(),
			TokenExpiresAt: time.Now().Add(interval),
		})
		time.Sleep(interval * 2)
		synctest.Wait()
		_, _, err := inmem.Get("key")
		if err == nil {
			t.Error("item was expected to be expired")
		}
	})
}
