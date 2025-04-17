package internal

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net/netip"
	"strings"
	"sync"
	"time"

	coreruleset "github.com/corazawaf/coraza-coreruleset/v4"
	"github.com/corazawaf/coraza/v3"
	"github.com/corazawaf/coraza/v3/types"
	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
	"github.com/jcchavezs/mergefs"
	"github.com/jcchavezs/mergefs/io"
	"github.com/rs/zerolog"
	"istio.io/istio/pkg/cache"
)

type AppConfig struct {
	Directives     string
	ResponseCheck  bool
	Logger         zerolog.Logger
	TransactionTTL time.Duration
}

type Application struct {
	waf   coraza.WAF
	cache cache.ExpiringCache

	AppConfig
}

type transaction struct {
	tx types.Transaction
	m  sync.Mutex
}

type applicationRequest struct {
	SrcIp   netip.Addr
	SrcPort int64
	DstIp   netip.Addr
	DstPort int64
	Method  string
	ID      string
	Path    []byte
	Query   []byte
	Version string
	Headers []byte
	Body    []byte
}

func (a *Application) HandleRequest(ctx context.Context, writer *encoding.ActionWriter, message *encoding.Message) (err error) {
	k := encoding.AcquireKVEntry()
	// run defer via anonymous function to not directly evaluate the arguments.
	defer func() {
		encoding.ReleaseKVEntry(k)
	}()

	var req applicationRequest
	for message.KV.Next(k) {
		switch name := string(k.NameBytes()); name {
		case "src-ip":
			req.SrcIp = k.ValueAddr()
		case "src-port":
			req.SrcPort = k.ValueInt()
		case "dst-ip":
			req.DstIp = k.ValueAddr()
		case "dst-port":
			req.DstPort = k.ValueInt()
		case "method":
			req.Method = string(k.ValueBytes())
		case "path":
			// make a copy of the pointer and add a defer in case there is another entry
			currK := k
			// run defer via anonymous function to not directly evaluate the arguments.
			defer func() {
				encoding.ReleaseKVEntry(currK)
			}()

			req.Path = currK.ValueBytes()

			// acquire a new kv entry to continue reading other message values.
			k = encoding.AcquireKVEntry()
		case "query":
			// make a copy of the pointer and add a defer in case there is another entry
			currK := k
			// run defer via anonymous function to not directly evaluate the arguments.
			defer func() {
				encoding.ReleaseKVEntry(currK)
			}()

			req.Query = currK.ValueBytes()
			// acquire a new kv entry to continue reading other message values.
			k = encoding.AcquireKVEntry()
		case "version":
			req.Version = string(k.ValueBytes())
		case "headers":
			// make a copy of the pointer and add a defer in case there is another entry
			currK := k
			// run defer via anonymous function to not directly evaluate the arguments.
			defer func() {
				encoding.ReleaseKVEntry(currK)
			}()

			req.Headers = currK.ValueBytes()
			// acquire a new kv entry to continue reading other message values.
			k = encoding.AcquireKVEntry()
		case "body":
			// make a copy of the pointer and add a defer in case there is another entry
			currK := k
			// run defer via anonymous function to not directly evaluate the arguments.
			defer func() {
				encoding.ReleaseKVEntry(currK)
			}()

			req.Body = currK.ValueBytes()
			// acquire a new kv entry to continue reading other message values.
			k = encoding.AcquireKVEntry()
		case "id":
			req.ID = string(k.ValueBytes())
		default:
			a.Logger.Debug().Str("name", name).Msg("unknown kv entry")
		}
	}

	// Check if we have received an id from haproxy
	if len(req.ID) == 0 {
		const idLength = 16
		var sb strings.Builder
		sb.Grow(idLength)
		for i := 0; i < idLength; i++ {
			sb.WriteRune(rune('A' + rand.Intn(26)))
		}
		req.ID = sb.String()
	}

	tx := a.waf.NewTransactionWithID(req.ID)
	defer func() {
		if err == nil && a.ResponseCheck {
			a.cache.SetWithExpiration(tx.ID(), &transaction{tx: tx}, a.TransactionTTL)
			return
		}

		tx.ProcessLogging()
		if err := tx.Close(); err != nil {
			a.Logger.Error().Str("tx", tx.ID()).Err(err).Msg("failed to close transaction")
		}
	}()

	if err := writer.SetString(encoding.VarScopeTransaction, "id", tx.ID()); err != nil {
		return err
	}

	if tx.IsRuleEngineOff() {
		a.Logger.Warn().Msg("Rule engine is Off, Coraza is not going to process any rule")
		return nil
	}

	tx.ProcessConnection(req.SrcIp.String(), int(req.SrcPort), req.DstIp.String(), int(req.DstPort))

	{
		url := strings.Builder{}
		url.Write(req.Path)
		if req.Query != nil {
			url.WriteString("?")
			url.Write(req.Query)
		}

		tx.ProcessURI(url.String(), req.Method, "HTTP/"+req.Version)
	}

	if err := readHeaders(req.Headers, tx.AddRequestHeader); err != nil {
		return fmt.Errorf("reading headers: %v", err)
	}

	if it := tx.ProcessRequestHeaders(); it != nil {
		return ErrInterrupted{it}
	}

	switch it, _, err := tx.WriteRequestBody(req.Body); {
	case err != nil:
		return err
	case it != nil:
		return ErrInterrupted{it}
	}

	switch it, err := tx.ProcessRequestBody(); {
	case err != nil:
		return err
	case it != nil:
		return ErrInterrupted{it}
	}

	return nil
}

func readHeaders(headers []byte, callback func(key string, value string)) error {
	s := bufio.NewScanner(bytes.NewReader(headers))
	for s.Scan() {
		line := bytes.TrimSpace(s.Bytes())
		if len(line) == 0 {
			continue
		}

		kv := bytes.SplitN(line, []byte(":"), 2)
		if len(kv) != 2 {
			return fmt.Errorf("invalid header: %q", s.Text())
		}

		key, value := bytes.TrimSpace(kv[0]), bytes.TrimSpace(kv[1])

		callback(string(key), string(value))
	}

	return nil
}

type applicationResponse struct {
	ID      string
	Version string
	Status  int64
	Headers []byte
	Body    []byte
}

func (a *Application) HandleResponse(ctx context.Context, writer *encoding.ActionWriter, message *encoding.Message) (err error) {
	if !a.ResponseCheck {
		return fmt.Errorf("got response but response check is disabled")
	}

	k := encoding.AcquireKVEntry()
	// run defer via anonymous function to not directly evaluate the arguments.
	defer func() {
		encoding.ReleaseKVEntry(k)
	}()

	var res applicationResponse
	for message.KV.Next(k) {
		switch name := string(k.NameBytes()); name {
		case "id":
			res.ID = string(k.ValueBytes())
		case "version":
			res.Version = string(k.ValueBytes())
		case "status":
			res.Status = k.ValueInt()
		case "headers":
			// make a copy of the pointer and add a defer in case there is another entry
			currK := k
			// run defer via anonymous function to not directly evaluate the arguments.
			defer func() {
				encoding.ReleaseKVEntry(currK)
			}()

			res.Headers = currK.ValueBytes()
			// acquire a new kv entry to continue reading other message values.
			k = encoding.AcquireKVEntry()
		case "body":
			// make a copy of the pointer and add a defer in case there is another entry
			currK := k
			// run defer via anonymous function to not directly evaluate the arguments.
			defer func() {
				encoding.ReleaseKVEntry(currK)
			}()

			res.Body = currK.ValueBytes()
			// acquire a new kv entry to continue reading other message values.
			k = encoding.AcquireKVEntry()
		default:
			a.Logger.Debug().Str("name", name).Msg("unknown kv entry")
		}
	}

	if res.ID == "" {
		return fmt.Errorf("response id is empty")
	}

	cv, ok := a.cache.Get(res.ID)
	if !ok {
		return fmt.Errorf("transaction not found: %s", res.ID)
	}
	a.cache.Remove(res.ID)

	t := cv.(*transaction)
	if !t.m.TryLock() {
		return fmt.Errorf("transaction is already being deleted: %s", res.ID)
	}
	tx := t.tx

	defer func() {
		tx.ProcessLogging()
		if err := tx.Close(); err != nil {
			a.Logger.Error().Str("tx", tx.ID()).Err(err).Msg("failed to close transaction")
		}
	}()

	if tx.IsRuleEngineOff() {
		goto exit
	}

	if err := readHeaders(res.Headers, tx.AddResponseHeader); err != nil {
		return fmt.Errorf("reading headers: %v", err)
	}

	if it := tx.ProcessResponseHeaders(int(res.Status), "HTTP/"+res.Version); it != nil {
		return ErrInterrupted{it}
	}

	switch it, _, err := tx.WriteResponseBody(res.Body); {
	case err != nil:
		return err
	case it != nil:
		return ErrInterrupted{it}
	}

	switch it, err := tx.ProcessResponseBody(); {
	case err != nil:
		return err
	case it != nil:
		return ErrInterrupted{it}
	}

exit:
	return nil
}

func (a AppConfig) NewApplication() (*Application, error) {
	app := Application{
		AppConfig: a,
	}

	config := coraza.NewWAFConfig().
		WithDirectives(a.Directives).
		WithErrorCallback(app.logCallback).
		WithRootFS(mergefs.Merge(coreruleset.FS, io.OSFS))

	waf, err := coraza.NewWAF(config)
	if err != nil {
		return nil, err
	}
	app.waf = waf

	const defaultExpire = time.Second * 10
	const defaultEvictionInterval = time.Second * 1

	app.cache = cache.NewTTLWithCallback(defaultExpire, defaultEvictionInterval, func(key, value any) {
		// everytime a transaction runs into a timeout it gets closed.
		t := value.(*transaction)
		if !t.m.TryLock() {
			// We lost a race and the transaction is already somewhere in use.
			a.Logger.Info().Str("tx", t.tx.ID()).Msg("eviction called on currently used transaction")
			return
		}

		// Process Logging won't do anything if TX was already logged.
		t.tx.ProcessLogging()
		if err := t.tx.Close(); err != nil {
			a.Logger.Error().Err(err).Str("tx", t.tx.ID()).Msg("error closing transaction")
		}
	})

	return &app, nil
}

func (a *Application) logCallback(mr types.MatchedRule) {
	var l *zerolog.Event

	switch mr.Rule().Severity() {
	case types.RuleSeverityWarning:
		l = a.Logger.Warn()
	case types.RuleSeverityNotice,
		types.RuleSeverityInfo:
		l = a.Logger.Info()
	case types.RuleSeverityDebug:
		l = a.Logger.Debug()
	default:
		l = a.Logger.Error()
	}
	l.Msg(mr.ErrorLog())
}

type ErrInterrupted struct {
	Interruption *types.Interruption
}

func (e ErrInterrupted) Error() string {
	return fmt.Sprintf("interrupted with status %d and action %s", e.Interruption.Status, e.Interruption.Action)
}

func (e ErrInterrupted) Is(target error) bool {
	t, ok := target.(*ErrInterrupted)
	if !ok {
		return false
	}
	return e.Interruption == t.Interruption
}
