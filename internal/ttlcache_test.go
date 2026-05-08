// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// pollUntil repeatedly checks condition until it returns true or deadline is exceeded.
// Returns true if condition was met, false if deadline was reached.
func pollUntil(deadline time.Time, interval time.Duration, condition func() bool) bool {
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(interval)
	}
	return condition()
}

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

	deadline := time.Now().Add(100 * time.Millisecond)
	expired := pollUntil(deadline, time.Millisecond, func() bool {
		_, ok := c.Get("key")
		return !ok
	})

	if !expired {
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

	c.SetWithExpiration("a", 1, 10*time.Millisecond)
	c.SetWithExpiration("b", 2, 10*time.Millisecond)

	deadline := time.Now().Add(200 * time.Millisecond)
	bothEvicted := pollUntil(deadline, 5*time.Millisecond, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return evicted["a"] == 1 && evicted["b"] == 2
	})

	c.stop()

	if !bothEvicted {
		mu.Lock()
		defer mu.Unlock()
		t.Errorf("expected both keys to be evicted: a=%v, b=%v", evicted["a"], evicted["b"])
	}
}

func TestTTLCache_EvictionCallbackNotCalledAfterRemove(t *testing.T) {
	var called atomic.Bool
	c := newTTLCache(10*time.Millisecond, func(_, _ any) {
		called.Store(true)
	})

	c.SetWithExpiration("key", "val", 10*time.Millisecond)
	c.Remove("key")

	// Wait for at least one eviction cycle to pass
	deadline := time.Now().Add(100 * time.Millisecond)
	pollUntil(deadline, 5*time.Millisecond, func() bool {
		return false // Just wait for the deadline
	})

	c.stop()

	if called.Load() {
		t.Fatal("eviction callback should not be called for manually removed key")
	}
}

func TestTTLCache_GetEagerExpiry(t *testing.T) {
	var mu sync.Mutex
	evictedKeys := []any{}

	c := newTTLCache(time.Minute, func(k, _ any) {
		mu.Lock()
		evictedKeys = append(evictedKeys, k)
		mu.Unlock()
	})
	defer c.stop()

	c.SetWithExpiration("key", "value", time.Millisecond)

	// Poll until the key expires when accessed via Get
	deadline := time.Now().Add(100 * time.Millisecond)
	keyExpired := pollUntil(deadline, time.Millisecond, func() bool {
		_, ok := c.Get("key")
		return !ok
	})

	if !keyExpired {
		t.Fatal("expected expired key to be absent")
	}

	// Poll until the eviction callback is invoked (it's async now)
	deadline = time.Now().Add(100 * time.Millisecond)
	callbackInvoked := pollUntil(deadline, time.Millisecond, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(evictedKeys) == 1
	})

	if !callbackInvoked {
		mu.Lock()
		count := len(evictedKeys)
		mu.Unlock()
		t.Fatalf("expected eviction callback called exactly once, got %d", count)
	}

	// A second Get should not trigger the callback again (entry was deleted).
	_, ok := c.Get("key")
	if ok {
		t.Fatal("expected key to remain absent")
	}

	// Give a bit of time for any spurious callback (shouldn't happen)
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	count := len(evictedKeys)
	mu.Unlock()
	if count != 1 {
		t.Fatalf("expected eviction callback still called exactly once, got %d", count)
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
