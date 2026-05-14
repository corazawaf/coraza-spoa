package internal

import (
	"context"
	"testing"
	"time"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
	"github.com/rs/zerolog"
)

func buildMessage(t *testing.T, name string, writeKV func(*encoding.KVWriter) error, kvCount byte) *encoding.Message {
	t.Helper()

	kvBuf := make([]byte, 4096)
	kvWriter := encoding.NewKVWriter(kvBuf, 0)
	if err := writeKV(kvWriter); err != nil {
		t.Fatal(err)
	}
	kvPayload := kvWriter.Bytes()

	msgBuf := make([]byte, 4096)
	off := 0
	n, err := encoding.PutVarint(msgBuf[off:], uint64(len(name)))
	if err != nil {
		t.Fatal(err)
	}
	off += n
	off += copy(msgBuf[off:], []byte(name))
	msgBuf[off] = kvCount
	off++
	off += copy(msgBuf[off:], kvPayload)

	scanner := encoding.NewMessageScanner(msgBuf[:off])
	msg := encoding.AcquireMessage()
	if !scanner.Next(msg) {
		t.Fatal(scanner.Error())
	}
	return msg
}

func TestAgentHandleSPOA_ResponseWithoutApp_DoesNotPanic(t *testing.T) {
	app := newTestApp(t)
	tx := app.waf.NewTransactionWithID("response-no-app")
	app.cache.SetWithExpiration(tx.ID(), &transaction{tx: tx}, 10*time.Second)

	a := &Agent{
		Context:            context.Background(),
		DefaultApplication: nil,
		Applications: map[string]*Application{
			"ftw": app,
		},
		Logger: zerolog.Nop(),
	}

	msg := buildMessage(t, "coraza-res", func(w *encoding.KVWriter) error {
		if err := w.SetString("id", tx.ID()); err != nil {
			return err
		}
		if err := w.SetString("version", "1.1"); err != nil {
			return err
		}
		return w.SetInt32("status", 200)
	}, 3)
	defer encoding.ReleaseMessage(msg)

	writer := encoding.NewActionWriter(make([]byte, 4096), 0)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("HandleSPOE panicked: %v", r)
		}
	}()

	a.HandleSPOE(context.Background(), writer, msg)

	if _, ok := app.cache.Get(tx.ID()); ok {
		t.Fatalf("expected transaction %q to be removed from cache after response handling", tx.ID())
	}
}

func TestSingleApplication(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		if got := singleApplication(map[string]*Application{}); got != nil {
			t.Fatal("expected nil for empty map")
		}
	})

	t.Run("single app", func(t *testing.T) {
		only := &Application{}
		if got := singleApplication(map[string]*Application{"only": only}); got != only {
			t.Fatal("expected sole application")
		}
	})

	t.Run("multiple apps", func(t *testing.T) {
		a := &Application{}
		b := &Application{}
		if got := singleApplication(map[string]*Application{"a": a, "b": b}); got != nil {
			t.Fatal("expected nil when map has multiple applications")
		}
	})
}
