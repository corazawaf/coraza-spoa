package internal

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/corazawaf/coraza/v3/types"
	"github.com/rs/zerolog"
)

const minimalDirectives = `SecRuleEngine On`

func newTestApplication(t *testing.T) *Application {
	t.Helper()

	cfg := AppConfig{
		Directives:     minimalDirectives,
		ResponseCheck:  true,
		Logger:         zerolog.New(os.Stderr).With().Timestamp().Logger(),
		TransactionTTL: 10 * time.Second,
	}

	app, err := cfg.NewApplication()
	if err != nil {
		t.Fatalf("failed to create application: %v", err)
	}
	return app
}

func newTestTransaction(t *testing.T, app *Application) types.Transaction {
	t.Helper()

	tx := app.waf.NewTransaction()
	tx.ProcessConnection("127.0.0.1", 12345, "127.0.0.1", 80)
	tx.ProcessURI("/test", "GET", "HTTP/1.1")
	tx.AddRequestHeader("host", "localhost")
	tx.ProcessRequestHeaders()
	return tx
}

func TestHandleResponseDetectOnly_ReturnsImmediately(t *testing.T) {
	app := newTestApplication(t)
	tx := newTestTransaction(t, app)

	res := applicationResponse{
		ID:         tx.ID(),
		Version:    "1.1",
		Status:     200,
		Headers:    []byte("content-type: text/html\r\n"),
		Body:       []byte("<html>ok</html>"),
		DetectOnly: true,
	}

	err := app.handleResponseDetectOnly(res, tx)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	app.DrainDetectOnly()
}

func TestHandleResponseDetectOnly_BackgroundEvalCompletes(t *testing.T) {
	app := newTestApplication(t)
	tx := newTestTransaction(t, app)

	res := applicationResponse{
		ID:         tx.ID(),
		Version:    "1.1",
		Status:     200,
		Headers:    []byte("content-type: text/html\r\n"),
		Body:       []byte("<html>ok</html>"),
		DetectOnly: true,
	}

	if err := app.handleResponseDetectOnly(res, tx); err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	done := make(chan struct{})
	go func() {
		app.DrainDetectOnly()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("DrainDetectOnly timed out: background evaluation did not complete")
	}
}

func TestHandleResponseDetectOnly_Concurrent(t *testing.T) {
	app := newTestApplication(t)

	var completed atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tx := newTestTransaction(t, app)
			res := applicationResponse{
				ID:         tx.ID(),
				Version:    "1.1",
				Status:     200,
				Headers:    []byte("server: test\r\n"),
				Body:       []byte("body"),
				DetectOnly: true,
			}
			if err := app.handleResponseDetectOnly(res, tx); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			completed.Add(1)
		}()
	}

	wg.Wait()
	app.DrainDetectOnly()

	if got := completed.Load(); got != 64 {
		t.Errorf("expected 64 completions, got %d", got)
	}
}

func TestEvaluateResponse_ProcessesAllPhases(t *testing.T) {
	app := newTestApplication(t)
	tx := newTestTransaction(t, app)

	headers := []byte("content-type: text/html\r\nserver: Apache\r\n")
	body := []byte("<html>response body</html>")

	app.evaluateResponse(tx, 200, "1.1", headers, body)
}

func TestEvaluateResponse_RuleEngineOff(t *testing.T) {
	cfg := AppConfig{
		Directives:     `SecRuleEngine Off`,
		ResponseCheck:  true,
		Logger:         zerolog.New(os.Stderr).With().Timestamp().Logger(),
		TransactionTTL: 10 * time.Second,
	}

	app, err := cfg.NewApplication()
	if err != nil {
		t.Fatalf("failed to create application: %v", err)
	}

	tx := app.waf.NewTransaction()
	app.evaluateResponse(tx, 200, "1.1", nil, nil)
}

func TestEvaluateResponse_InvalidHeaders(t *testing.T) {
	app := newTestApplication(t)
	tx := newTestTransaction(t, app)

	// Malformed header (no colon separator) triggers error path.
	headers := []byte("invalid-header-no-colon\r\n")
	body := []byte("body")

	app.evaluateResponse(tx, 200, "1.1", headers, body)
}

func TestEvaluateResponse_NilHeadersAndBody(t *testing.T) {
	app := newTestApplication(t)
	tx := newTestTransaction(t, app)

	app.evaluateResponse(tx, 200, "1.1", nil, nil)
}

func TestDetectOnlyWithMaliciousResponse(t *testing.T) {
	directives := `
SecRuleEngine On
SecResponseBodyAccess On
SecRule RESPONSE_BODY "@contains secret-token" "id:900100,phase:4,deny,status:403,msg:'Data leak detected'"
`
	cfg := AppConfig{
		Directives:     directives,
		ResponseCheck:  true,
		Logger:         zerolog.New(zerolog.TestWriter{T: t}).With().Timestamp().Logger(),
		TransactionTTL: 10 * time.Second,
	}

	app, err := cfg.NewApplication()
	if err != nil {
		t.Fatalf("failed to create application: %v", err)
	}

	tx := app.waf.NewTransaction()
	tx.ProcessConnection("127.0.0.1", 12345, "127.0.0.1", 80)
	tx.ProcessURI("/test", "GET", "HTTP/1.1")
	tx.AddRequestHeader("host", "localhost")
	tx.ProcessRequestHeaders()

	res := applicationResponse{
		ID:         tx.ID(),
		Version:    "1.1",
		Status:     200,
		Headers:    []byte("content-type: text/plain\r\n"),
		Body:       []byte("here is a secret-token leaked"),
		DetectOnly: true,
	}

	err = app.handleResponseDetectOnly(res, tx)
	if err != nil {
		t.Fatalf("detect-only should not return error, got: %v", err)
	}

	app.DrainDetectOnly()
}

func TestDrainDetectOnly_EmptyPool(t *testing.T) {
	app := newTestApplication(t)

	done := make(chan struct{})
	go func() {
		app.DrainDetectOnly()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("DrainDetectOnly blocked with no in-flight evaluations")
	}
}

