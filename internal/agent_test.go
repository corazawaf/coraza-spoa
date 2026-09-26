package internal

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/rs/zerolog"
)

// buildSPOEMessage encodes a SPOE message named name whose KV entries are
// written by kv, and returns it decoded as HandleSPOE receives it.
func buildSPOEMessage(t *testing.T, name string, kv func(*encoding.KVWriter) error) *encoding.Message {
	t.Helper()

	kvBuf := make([]byte, 4096)
	kw := encoding.NewKVWriter(kvBuf, 0)
	if err := kw.SetString("app", "default"); err != nil {
		t.Fatal(err)
	}
	if err := kv(kw); err != nil {
		t.Fatal(err)
	}
	kvBytes := kvBuf[:kw.Off()]

	// Count the entries by scanning them back, as HAProxy sends the count
	// ahead of the KV list.
	var count byte
	scanner := encoding.NewKVScanner(kvBytes, -1)
	k := encoding.AcquireKVEntry()
	defer encoding.ReleaseKVEntry(k)
	for scanner.Next(k) {
		count++
	}

	// name length fits a single-byte varint for the names used here.
	raw := append([]byte{byte(len(name))}, name...)
	raw = append(raw, count)
	raw = append(raw, kvBytes...)

	msg := encoding.AcquireMessage()
	t.Cleanup(func() { encoding.ReleaseMessage(msg) })
	if !encoding.NewMessageScanner(raw).Next(msg) {
		t.Fatal("failed to decode message")
	}
	return msg
}

func newTestAgent(t *testing.T, responseCheck bool) (*Agent, *Application) {
	t.Helper()
	app, err := AppConfig{
		ResponseCheck:  responseCheck,
		Logger:         zerolog.Nop(),
		TransactionTTL: 10 * time.Second,
	}.NewApplication()
	if err != nil {
		t.Fatal(err)
	}

	// A Nop logger would disable Logger.Panic too, hiding the regression
	// these tests guard against.
	a := &Agent{Context: context.Background(), Logger: zerolog.New(io.Discard)}
	a.ReplaceApplications(map[string]*Application{"default": app}, app)
	return a, app
}

func handleSPOE(t *testing.T, a *Agent, msg *encoding.Message) []byte {
	t.Helper()
	aw := encoding.NewActionWriter(make([]byte, 4096), 0)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("HandleSPOE panicked: %v", r)
		}
	}()
	a.HandleSPOE(context.Background(), aw, msg)
	return aw.Bytes()
}

func counterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		t.Fatal(err)
	}
	return m.GetCounter().GetValue()
}

func errorVarActions(t *testing.T, code int64) []byte {
	t.Helper()
	aw := encoding.NewActionWriter(make([]byte, 64), 0)
	if err := aw.SetInt64(encoding.VarScopeTransaction, "error", code); err != nil {
		t.Fatal(err)
	}
	return aw.Bytes()
}

func TestHandleSPOE_UncorrelatedResponseFailsClosed(t *testing.T) {
	tests := []struct {
		name                  string
		id                    string
		responseCheckDisabled bool
		setup                 func(t *testing.T, app *Application)
		reason                CorrelationFailure
	}{
		{
			name:   "no preceding coraza-req",
			id:     "never-requested",
			reason: ReasonNotFound,
		},
		{
			name:   "missing id",
			reason: ReasonMissingID,
		},
		{
			name: "transaction being closed",
			id:   "closing",
			setup: func(t *testing.T, app *Application) {
				tx := app.waf.NewTransactionWithID("closing")
				t.Cleanup(func() {
					if err := tx.Close(); err != nil {
						t.Errorf("closing transaction: %v", err)
					}
				})
				cached := &transaction{tx: tx}
				// Simulate TTL eviction holding the transaction.
				cached.m.Lock()
				app.cache.SetWithExpiration(tx.ID(), cached, 10*time.Second)
			},
			reason: ReasonClosing,
		},
		{
			name:                  "response check disabled",
			id:                    "whatever",
			responseCheckDisabled: true,
			reason:                ReasonResponseCheckDisabled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, app := newTestAgent(t, !tt.responseCheckDisabled)
			if tt.setup != nil {
				tt.setup(t, app)
			}

			counter := responseUncorrelatedTotal.WithLabelValues(tt.reason.String())
			before := counterValue(t, counter)

			msg := buildSPOEMessage(t, "coraza-res", func(kw *encoding.KVWriter) error {
				if tt.id != "" {
					if err := kw.SetString("id", tt.id); err != nil {
						return err
					}
				}
				if err := kw.SetString("version", "1.1"); err != nil {
					return err
				}
				return kw.SetInt32("status", 200)
			})

			got := handleSPOE(t, a, msg)
			if want := errorVarActions(t, tt.reason.ErrorCode()); !bytes.Equal(got, want) {
				t.Errorf("actions = %x, want only the error var %x", got, want)
			}
			if d := counterValue(t, counter) - before; d != 1 {
				t.Errorf("coraza_response_uncorrelated_total{reason=%q} increased by %v, want 1", tt.reason, d)
			}
		})
	}
}
