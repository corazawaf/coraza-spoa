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
type ttlCache struct {
	mu               sync.Mutex
	entries          map[any]*ttlEntry
	evictionCallback func(key, value any)
	stopCh           chan struct{}
}

func newTTLCache(evictionInterval time.Duration, onEvict func(key, value any)) *ttlCache {
	c := &ttlCache{
		entries:          make(map[any]*ttlEntry),
		evictionCallback: onEvict,
		stopCh:           make(chan struct{}),
	}
	go c.evictLoop(evictionInterval)
	return c
}

func (c *ttlCache) evictLoop(interval time.Duration) {
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
	now := time.Now()
	c.mu.Lock()
	var expired []struct{ k, v any }
	for k, e := range c.entries {
		if now.After(e.expiresAt) {
			expired = append(expired, struct{ k, v any }{k, e.value})
			delete(c.entries, k)
		}
	}
	c.mu.Unlock()

	for _, pair := range expired {
		c.evictionCallback(pair.k, pair.v)
	}
}

func (c *ttlCache) SetWithExpiration(key, value any, ttl time.Duration) {
	c.mu.Lock()
	c.entries[key] = &ttlEntry{value: value, expiresAt: time.Now().Add(ttl)}
	c.mu.Unlock()
}

func (c *ttlCache) Get(key any) (any, bool) {
	c.mu.Lock()
	e, ok := c.entries[key]
	c.mu.Unlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.value, true
}

func (c *ttlCache) Remove(key any) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

func (c *ttlCache) stop() {
	close(c.stopCh)
}
