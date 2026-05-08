// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"sync"
	"time"
)

type ttlEntry struct {
	value     any
	expiresAt time.Time
}

// ttlCache is a thread-safe cache with per-entry TTL and an eviction callback.
// The eviction callback is invoked asynchronously in a separate goroutine
// without holding any cache locks. This prevents deadlock if the callback
// calls stop() and ensures stop() can complete without waiting on itself.
type ttlCache struct {
	mu               sync.Mutex
	entries          map[any]*ttlEntry
	evictionCallback func(key, value any)
	stopCh           chan struct{}
	stopOnce         sync.Once
	done             chan struct{}
}

func newTTLCache(evictionInterval time.Duration, onEvict func(key, value any)) *ttlCache {
	if evictionInterval <= 0 {
		panic("ttlcache: evictionInterval must be positive")
	}
	c := &ttlCache{
		entries:          make(map[any]*ttlEntry),
		evictionCallback: onEvict,
		stopCh:           make(chan struct{}),
		done:             make(chan struct{}),
	}
	go c.evictLoop(evictionInterval)
	return c
}

func (c *ttlCache) evictLoop(interval time.Duration) {
	defer close(c.done)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.evictExpired()
		case <-c.stopCh:
			return
		}
	}
}

func (c *ttlCache) evictExpired() {
	c.mu.Lock()
	now := time.Now()
	var expired []struct{ k, v any }
	for k, e := range c.entries {
		if !now.Before(e.expiresAt) {
			expired = append(expired, struct{ k, v any }{k, e.value})
			delete(c.entries, k)
		}
	}
	c.mu.Unlock()

	for _, pair := range expired {
		go c.evictionCallback(pair.k, pair.v)
	}
}

func (c *ttlCache) SetWithExpiration(key, value any, ttl time.Duration) {
	c.mu.Lock()
	c.entries[key] = &ttlEntry{value: value, expiresAt: time.Now().Add(ttl)}
	c.mu.Unlock()
}

func (c *ttlCache) Get(key any) (any, bool) {
	c.mu.Lock()
	now := time.Now()
	e, ok := c.entries[key]
	if !ok {
		c.mu.Unlock()
		return nil, false
	}
	if !now.Before(e.expiresAt) {
		value := e.value
		delete(c.entries, key)
		c.mu.Unlock()
		go c.evictionCallback(key, value)
		return nil, false
	}
	value := e.value
	c.mu.Unlock()
	return value, true
}

func (c *ttlCache) Remove(key any) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

func (c *ttlCache) stop() {
	c.stopOnce.Do(func() {
		close(c.stopCh)
	})
	<-c.done
}
