package internal

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/rs/zerolog"
)

// Rules used by metrics tests:
//
//	190001: attack range, denies /attack.
//	190050: attack range, passes /probe (rule_triggers without disrupting).
//	150001: outside attack range, passes /noisy (must be filtered out).
const testDirectives = `
SecRuleEngine On
SecRequestBodyAccess On
SecResponseBodyAccess On
SecRule REQUEST_URI "@contains /attack" "id:190001,phase:1,deny,severity:CRITICAL,msg:'test attack'"
SecRule REQUEST_URI "@contains /probe" "id:190050,phase:1,pass,severity:WARNING,msg:'test probe'"
SecRule REQUEST_URI "@contains /noisy" "id:150001,phase:1,pass,severity:WARNING,msg:'test noise'"
`

func newAgentForMetrics(t *testing.T, responseCheck bool) *Agent {
	t.Helper()
	cfg := AppConfig{
		Directives:     testDirectives,
		ResponseCheck:  responseCheck,
		Logger:         zerolog.Nop(),
		TransactionTTL: 10 * time.Second,
	}
	app, err := cfg.NewApplication()
	if err != nil {
		t.Fatalf("NewApplication: %v", err)
	}
	agent := &Agent{
		Context: context.Background(),
		Logger:  zerolog.Nop(),
	}
	agent.ReplaceApplications(map[string]*Application{"sample_app": app}, app)
	return agent
}

func histogramSampleCount(t *testing.T, h prometheus.Histogram) uint64 {
	t.Helper()
	m := &dto.Metric{}
	if err := h.(prometheus.Metric).Write(m); err != nil {
		t.Fatalf("write histogram: %v", err)
	}
	return m.GetHistogram().GetSampleCount()
}

func mustKV(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// buildSPOEMessage assembles an in-memory SPOE frame and parses it back
// via MessageScanner, mirroring the real wire path. The returned Message
// holds slices into the backing buffer.
func buildSPOEMessage(t *testing.T, name string, write func(kw *encoding.KVWriter) int) *encoding.Message {
	t.Helper()
	buf := make([]byte, 16384)

	nameEnd, err := encoding.PutBytes(buf, []byte(name))
	if err != nil {
		t.Fatalf("encode name: %v", err)
	}
	countOff := nameEnd
	kw := encoding.NewKVWriter(buf, nameEnd+1)
	n := write(kw)
	if n < 0 || n > 255 {
		t.Fatalf("invalid kv count %d", n)
	}
	buf[countOff] = byte(n)

	scanner := encoding.NewMessageScanner(buf[:kw.Off()])
	msg := &encoding.Message{}
	if !scanner.Next(msg) {
		t.Fatalf("MessageScanner.Next: %v", scanner.Error())
	}
	return msg
}

func reqMessage(t *testing.T, app, path, txID string) *encoding.Message {
	return buildSPOEMessage(t, "coraza-req", func(kw *encoding.KVWriter) int {
		mustKV(t, kw.SetString("app", app))
		mustKV(t, kw.SetAddr("src-ip", netip.MustParseAddr("127.0.0.1")))
		mustKV(t, kw.SetInt32("src-port", 12345))
		mustKV(t, kw.SetAddr("dst-ip", netip.MustParseAddr("127.0.0.1")))
		mustKV(t, kw.SetInt32("dst-port", 8080))
		mustKV(t, kw.SetString("method", "GET"))
		mustKV(t, kw.SetBinary("path", []byte(path)))
		mustKV(t, kw.SetBinary("query", nil))
		mustKV(t, kw.SetString("version", "1.1"))
		mustKV(t, kw.SetBinary("headers", []byte("host: test\n")))
		mustKV(t, kw.SetBinary("body", nil))
		mustKV(t, kw.SetString("id", txID))
		return 12
	})
}

func resMessage(t *testing.T, app, txID string) *encoding.Message {
	return buildSPOEMessage(t, "coraza-res", func(kw *encoding.KVWriter) int {
		mustKV(t, kw.SetString("app", app))
		mustKV(t, kw.SetString("id", txID))
		mustKV(t, kw.SetString("version", "1.1"))
		mustKV(t, kw.SetInt32("status", 200))
		mustKV(t, kw.SetBinary("headers", []byte("content-type: text/plain\n")))
		mustKV(t, kw.SetBinary("body", nil))
		return 6
	})
}

func sendSPOE(t *testing.T, a *Agent, msg *encoding.Message) {
	t.Helper()
	aw := encoding.NewActionWriter(make([]byte, 4096), 0)
	a.HandleSPOE(context.Background(), aw, msg)
}

func TestActionsTotal_AllowOnRequestWhenNoResponseCheck(t *testing.T) {
	a := newAgentForMetrics(t, false)

	c := actionsTotal.WithLabelValues("allow", "sample_app")
	before := testutil.ToFloat64(c)

	sendSPOE(t, a, reqMessage(t, "sample_app", "/clean", "tx-allow-1"))
	if got := testutil.ToFloat64(c) - before; got != 1 {
		t.Fatalf("expected +1 increment, got %v", got)
	}
}

func TestActionsTotal_AllowOnlyAtResponseWhenResponseCheck(t *testing.T) {
	a := newAgentForMetrics(t, true)
	c := actionsTotal.WithLabelValues("allow", "sample_app")
	before := testutil.ToFloat64(c)

	sendSPOE(t, a, reqMessage(t, "sample_app", "/clean", "tx-allow-2"))
	if got := testutil.ToFloat64(c) - before; got != 0 {
		t.Fatalf("expected +0 after request phase (verdict not final), got %v", got)
	}

	sendSPOE(t, a, resMessage(t, "sample_app", "tx-allow-2"))
	if got := testutil.ToFloat64(c) - before; got != 1 {
		t.Fatalf("expected +1 after response phase, got %v", got)
	}
}

func TestActionsTotal_DenyOnInterruption(t *testing.T) {
	a := newAgentForMetrics(t, true)
	c := actionsTotal.WithLabelValues("deny", "sample_app")
	before := testutil.ToFloat64(c)

	sendSPOE(t, a, reqMessage(t, "sample_app", "/attack", "tx-deny-1"))
	if got := testutil.ToFloat64(c) - before; got != 1 {
		t.Fatalf("expected +1 deny increment, got %v", got)
	}
}

func TestActionsTotal_DefaultAppNameOnFallback(t *testing.T) {
	// Fallback to DefaultApplication labels with the default's configured
	// name, not the requested one. Bounds cardinality when HAProxy SPOE
	// args come from request data (hdr(host) and similar).
	a := newAgentForMetrics(t, false)
	defaultLabel := actionsTotal.WithLabelValues("allow", "sample_app")
	requestedLabel := actionsTotal.WithLabelValues("allow", "unknown-app")
	defaultBefore := testutil.ToFloat64(defaultLabel)
	requestedBefore := testutil.ToFloat64(requestedLabel)

	sendSPOE(t, a, reqMessage(t, "unknown-app", "/clean", "tx-fallback-1"))
	if got := testutil.ToFloat64(defaultLabel) - defaultBefore; got != 1 {
		t.Fatalf("expected default's name in label (+1), got %v", got)
	}
	if got := testutil.ToFloat64(requestedLabel) - requestedBefore; got != 0 {
		t.Fatalf("requested name must not appear in label, got %v", got)
	}
}

func TestRuleTriggersTotal_OnlyAttackRangeRulesCounted(t *testing.T) {
	a := newAgentForMetrics(t, false)

	attack := ruleTriggersTotal.WithLabelValues("190050", "warning")
	nonAttack := ruleTriggersTotal.WithLabelValues("150001", "warning")
	aBefore := testutil.ToFloat64(attack)
	nBefore := testutil.ToFloat64(nonAttack)

	sendSPOE(t, a, reqMessage(t, "sample_app", "/probe", "tx-rt-1"))
	if got := testutil.ToFloat64(attack) - aBefore; got != 1 {
		t.Errorf("attack-range rule 190050 expected +1, got %v", got)
	}

	sendSPOE(t, a, reqMessage(t, "sample_app", "/noisy", "tx-rt-2"))
	if got := testutil.ToFloat64(nonAttack) - nBefore; got != 0 {
		t.Errorf("non-attack rule 150001 must be filtered out by isAttackRule, got %v", got)
	}
}

func TestRuleTriggersTotal_NoDoubleCountWithResponseCheck(t *testing.T) {
	a := newAgentForMetrics(t, true)
	c := ruleTriggersTotal.WithLabelValues("190050", "warning")
	before := testutil.ToFloat64(c)

	sendSPOE(t, a, reqMessage(t, "sample_app", "/probe", "tx-rt-3"))
	if got := testutil.ToFloat64(c) - before; got != 0 {
		t.Errorf("expected +0 after request phase (final=false), got %v", got)
	}

	sendSPOE(t, a, resMessage(t, "sample_app", "tx-rt-3"))
	if got := testutil.ToFloat64(c) - before; got != 1 {
		t.Errorf("expected +1 after both phases (no double count), got %v", got)
	}
}

func TestExportWAFMetrics_OrphanedTransactionEvicted(t *testing.T) {
	// Build an Application with a very short TTL so we can force eviction
	// inside the test instead of waiting for the 1s eviction loop tick.
	cfg := AppConfig{
		Directives:     testDirectives,
		ResponseCheck:  true,
		Logger:         zerolog.Nop(),
		TransactionTTL: 5 * time.Millisecond,
	}
	app, err := cfg.NewApplication()
	if err != nil {
		t.Fatalf("NewApplication: %v", err)
	}
	a := &Agent{
		Context: context.Background(),
		Logger:  zerolog.Nop(),
	}
	a.ReplaceApplications(map[string]*Application{"sample_app": app}, app)

	rules := ruleTriggersTotal.WithLabelValues("190050", "warning")
	allow := actionsTotal.WithLabelValues("allow", "sample_app")
	rulesBefore := testutil.ToFloat64(rules)
	allowBefore := testutil.ToFloat64(allow)
	scoreBefore := histogramSampleCount(t, anomalyScore)

	// Request phase only: the rule matches and the tx is cached awaiting
	// a response phase that will never arrive.
	sendSPOE(t, a, reqMessage(t, "sample_app", "/probe", "tx-orphan-1"))
	if got := testutil.ToFloat64(rules) - rulesBefore; got != 0 {
		t.Fatalf("request phase should not increment rule_triggers under ResponseCheck (got %v)", got)
	}
	if got := histogramSampleCount(t, anomalyScore) - scoreBefore; got != 0 {
		t.Fatalf("request phase should not observe anomaly_score under ResponseCheck (got %d)", got)
	}

	// Wait past the TTL, then drive eviction synchronously.
	time.Sleep(20 * time.Millisecond)
	app.cache.evictExpired()

	// The eviction callback runs in its own goroutine, so poll briefly.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if testutil.ToFloat64(rules)-rulesBefore == 1 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if got := testutil.ToFloat64(rules) - rulesBefore; got != 1 {
		t.Fatalf("expected rule_triggers +1 from eviction within 1s, got %v", got)
	}

	// anomaly_score must also be observed once via the eviction path,
	// otherwise observations for orphan transactions silently vanish.
	if got := histogramSampleCount(t, anomalyScore) - scoreBefore; got != 1 {
		t.Errorf("expected anomaly_score +1 from eviction, got %d", got)
	}

	// Orphans never reached a verdict - must not be mislabelled as "allow".
	// Guards against a future change that hooks actionsTotal into eviction.
	if got := testutil.ToFloat64(allow) - allowBefore; got != 0 {
		t.Errorf("orphan eviction must not increment actions_total{allow}, got %v", got)
	}
}

func TestHandleSPOEDuration_ObservedOncePerCall(t *testing.T) {
	a := newAgentForMetrics(t, true)
	before := histogramSampleCount(t, handleSPOEDuration)

	sendSPOE(t, a, reqMessage(t, "sample_app", "/clean", "tx-spoe-dur-1"))
	if got := histogramSampleCount(t, handleSPOEDuration) - before; got != 1 {
		t.Fatalf("expected +1 after request phase, got %d", got)
	}

	sendSPOE(t, a, resMessage(t, "sample_app", "tx-spoe-dur-1"))
	if got := histogramSampleCount(t, handleSPOEDuration) - before; got != 2 {
		t.Fatalf("expected +2 after response phase (both phases are separate SPOE calls), got %d", got)
	}
}

func TestHandleSPOEDuration_ObservedOnInterruption(t *testing.T) {
	a := newAgentForMetrics(t, true)
	before := histogramSampleCount(t, handleSPOEDuration)

	// Interruption causes early return; the deferred ObserveDuration must
	// still fire. Pins the defer placement at the top of HandleSPOE.
	sendSPOE(t, a, reqMessage(t, "sample_app", "/attack", "tx-spoe-dur-int"))
	if got := histogramSampleCount(t, handleSPOEDuration) - before; got != 1 {
		t.Fatalf("expected +1 on interrupted call, got %d", got)
	}
}

func TestHandleSPOEDuration_ObservedOnUnknownMessage(t *testing.T) {
	a := newAgentForMetrics(t, false)
	before := histogramSampleCount(t, handleSPOEDuration)

	// Unknown message names return early. The timer is created before the
	// switch, so the observation should still fire - guards against a
	// refactor that moves the timer below the switch.
	msg := buildSPOEMessage(t, "unknown-message", func(kw *encoding.KVWriter) int {
		mustKV(t, kw.SetString("app", "sample_app"))
		return 1
	})
	sendSPOE(t, a, msg)
	if got := histogramSampleCount(t, handleSPOEDuration) - before; got != 1 {
		t.Fatalf("expected +1 even on unknown message, got %d", got)
	}
}

func TestAnomalyScore_ObservedOncePerTransactionWithResponseCheck(t *testing.T) {
	a := newAgentForMetrics(t, true)
	before := histogramSampleCount(t, anomalyScore)

	sendSPOE(t, a, reqMessage(t, "sample_app", "/clean", "tx-as-1"))
	if got := histogramSampleCount(t, anomalyScore); got != before {
		t.Errorf("anomaly score should not be observed at request phase under ResponseCheck (before=%d got=%d)", before, got)
	}

	sendSPOE(t, a, resMessage(t, "sample_app", "tx-as-1"))
	if got := histogramSampleCount(t, anomalyScore); got != before+1 {
		t.Errorf("anomaly score should be observed exactly once after response phase (want %d, got %d)", before+1, got)
	}
}
