// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"sync"
	"testing"
	"time"
)

func TestTTLCache_SetAndGet(t *testing.T) {
	c := newTTLCache(time.Minute, func(_, _ any) {})
	defer c.stop()

	c.SetWithExpiration("key", "value", time.Minute)

	v, ok := c.Get("key")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if v.(string) != "value" {
		t.Fatalf("expected 'value', got %v", v)
	}
}

func TestTTLCache_MissingKey(t *testing.T) {
	c := newTTLCache(time.Minute, func(_, _ any) {})
	defer c.stop()

	_, ok := c.Get("missing")
	if ok {
		t.Fatal("expected key to be absent")
	}
}

func TestTTLCache_Remove(t *testing.T) {
	c := newTTLCache(time.Minute, func(_, _ any) {})
	defer c.stop()

	c.SetWithExpiration("key", "value", time.Minute)
	c.Remove("key")

	_, ok := c.Get("key")
	if ok {
		t.Fatal("expected key to be removed")
	}
}

func TestTTLCache_Expiry(t *testing.T) {
	c := newTTLCache(time.Minute, func(_, _ any) {})
	defer c.stop()

	c.SetWithExpiration("key", "value", time.Millisecond)
	time.Sleep(10 * time.Millisecond)

	_, ok := c.Get("key")
	if ok {
		t.Fatal("expected key to be expired")
	}
}

func TestTTLCache_EvictionCallback(t *testing.T) {
	var mu sync.Mutex
	evicted := map[any]any{}

	c := newTTLCache(10*time.Millisecond, func(k, v any) {
		mu.Lock()
		evicted[k] = v
		mu.Unlock()
	})
	defer c.stop()

	c.SetWithExpiration("a", 1, 10*time.Millisecond)
	c.SetWithExpiration("b", 2, 10*time.Millisecond)

	// Wait for eviction loop to fire
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if evicted["a"] != 1 {
		t.Errorf("expected 'a' to be evicted with value 1, got %v", evicted["a"])
	}
	if evicted["b"] != 2 {
		t.Errorf("expected 'b' to be evicted with value 2, got %v", evicted["b"])
	}
}

func TestTTLCache_EvictionCallbackNotCalledAfterRemove(t *testing.T) {
	called := false
	c := newTTLCache(10*time.Millisecond, func(_, _ any) {
		called = true
	})
	defer c.stop()

	c.SetWithExpiration("key", "val", 10*time.Millisecond)
	c.Remove("key")

	time.Sleep(100 * time.Millisecond)

	if called {
		t.Fatal("eviction callback should not be called for manually removed key")
	}
}

func TestTTLCache_Concurrent(t *testing.T) {
	c := newTTLCache(time.Millisecond*10, func(_, _ any) {})
	defer c.stop()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := n % 10
			c.SetWithExpiration(key, n, 50*time.Millisecond)
			c.Get(key)
			c.Remove(key)
		}(i)
	}
	wg.Wait()
}
