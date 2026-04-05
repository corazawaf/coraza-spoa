package internal

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
	"github.com/rs/zerolog"
)

func newTestApp(t *testing.T) *Application {
	t.Helper()
	app, err := AppConfig{
		Directives:     "",
		ResponseCheck:  true,
		Logger:         zerolog.Nop(),
		TransactionTTL: 10 * time.Second,
	}.NewApplication()
	if err != nil {
		t.Fatal(err)
	}
	return app
}

// buildDetectOnlyMessage creates a KV-encoded message with the fields
// required by HandleResponse in detect-only mode.
func buildDetectOnlyMessage(t *testing.T, txID string) (*encoding.ActionWriter, *encoding.Message) {
	t.Helper()

	kvBuf := make([]byte, 4096)
	kw := encoding.NewKVWriter(kvBuf, 0)
	if err := kw.SetString("id", txID); err != nil {
		t.Fatal(err)
	}
	if err := kw.SetString("version", "1.1"); err != nil {
		t.Fatal(err)
	}
	if err := kw.SetInt32("status", 200); err != nil {
		t.Fatal(err)
	}
	if err := kw.SetBool("detect-only", true); err != nil {
		t.Fatal(err)
	}

	scanner := encoding.NewKVScanner(kvBuf[:kw.Off()], 4)
	msg := &encoding.Message{KV: scanner}
	aw := encoding.NewActionWriter(make([]byte, 4096), 0)
	return aw, msg
}

// TestDrainDetectOnly_WaitsForInFlight verifies that DrainDetectOnly
// blocks until all in-flight detect-only goroutines complete.
func TestDrainDetectOnly_WaitsForInFlight(t *testing.T) {
	app := newTestApp(t)

	const n = 10
	// Simulate n in-flight detect-only goroutines.
	for i := 0; i < n; i++ {
		app.asyncMu.Lock()
		app.asyncWg.Add(1)
		app.asyncMu.Unlock()

		go func() {
			defer app.asyncWg.Done()
			time.Sleep(50 * time.Millisecond)
		}()
	}

	done := make(chan struct{})
	go func() {
		app.DrainDetectOnly()
		close(done)
	}()

	select {
	case <-done:
		// DrainDetectOnly returned after all goroutines finished.
	case <-time.After(5 * time.Second):
		t.Fatal("DrainDetectOnly did not return in time")
	}

	// Verify draining flag is set.
	app.asyncMu.Lock()
	if !app.draining {
		t.Error("expected draining to be true after DrainDetectOnly")
	}
	app.asyncMu.Unlock()
}

// TestDrainDetectOnly_FallbackToSync verifies that after DrainDetectOnly
// is called, detect-only requests fall back to synchronous evaluation.
func TestDrainDetectOnly_FallbackToSync(t *testing.T) {
	app := newTestApp(t)

	// Create a transaction and cache it so HandleResponse can find it.
	tx := app.waf.NewTransactionWithID("drain-sync-test")
	app.cache.SetWithExpiration(tx.ID(), &transaction{tx: tx}, 10*time.Second)

	// Drain first (no in-flight work, returns immediately).
	app.DrainDetectOnly()

	aw, msg := buildDetectOnlyMessage(t, tx.ID())

	// HandleResponse should execute synchronously (no goroutine spawned).
	err := app.HandleResponse(context.Background(), aw, msg)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify no async work was queued (WaitGroup counter should be zero
	// and return immediately).
	done := make(chan struct{})
	go func() {
		app.asyncWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Good — no pending async work.
	case <-time.After(time.Second):
		t.Fatal("asyncWg.Wait blocked — goroutine was spawned despite draining")
	}
}

// TestDrainDetectOnly_ConcurrentRace runs DrainDetectOnly concurrently
// with detect-only HandleResponse calls to verify there is no race
// between asyncWg.Add(1) and asyncWg.Wait().
// Run with: go test -race -run TestDrainDetectOnly_ConcurrentRace
func TestDrainDetectOnly_ConcurrentRace(t *testing.T) {
	app := newTestApp(t)

	const workers = 20
	var started sync.WaitGroup
	started.Add(workers)

	var asyncCount atomic.Int32

	// entered tracks how many workers have entered HandleResponse.
	var entered atomic.Int32

	// Launch workers that simulate detect-only requests.
	for i := 0; i < workers; i++ {
		txID := fmt.Sprintf("race-test-%d", i)
		tx := app.waf.NewTransactionWithID(txID)
		app.cache.SetWithExpiration(tx.ID(), &transaction{tx: tx}, 10*time.Second)

		go func(id string) {
			started.Done()
			started.Wait() // all workers start at the same time

			entered.Add(1)
			aw, msg := buildDetectOnlyMessage(t, id)
			err := app.HandleResponse(context.Background(), aw, msg)
			if err != nil {
				return
			}
			asyncCount.Add(1)
		}(tx.ID())
	}

	// Wait for all workers to be ready, then wait until at least some
	// have entered HandleResponse before draining.
	started.Wait()
	for entered.Load() == 0 {
		time.Sleep(time.Millisecond)
	}

	app.DrainDetectOnly()

	// After drain, all work (async or sync fallback) must be complete.
	// If the race existed, the -race detector would flag it here.
	t.Logf("completed requests: %d", asyncCount.Load())
}
