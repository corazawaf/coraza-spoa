package internal

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
	"github.com/dropmorepackets/haproxy-go/spop"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

type Agent struct {
	Context            context.Context
	DefaultApplication *Application
	Applications       map[string]*Application
	Logger             zerolog.Logger

	// defaultApplicationName caches the map key under which DefaultApplication
	// is stored in Applications. Maintained by ReplaceApplications so the hot
	// path in HandleSPOE can label the fallback-verdict metric in O(1).
	defaultApplicationName string

	mtx sync.RWMutex
}

func (a *Agent) Serve(l net.Listener) error {
	agent := spop.Agent{
		Handler:     a,
		BaseContext: a.Context,
	}

	return agent.Serve(l)
}

func (a *Agent) ReplaceApplications(newApps map[string]*Application, defaultApp *Application) {
	var defaultName string
	if defaultApp != nil {
		for name, app := range newApps {
			if app == defaultApp {
				defaultName = name
				break
			}
		}
	}
	a.mtx.Lock()
	a.Applications = newApps
	a.DefaultApplication = defaultApp
	a.defaultApplicationName = defaultName
	a.mtx.Unlock()
}

// DrainDetectOnly blocks until all in-flight detect-only evaluations
// complete across all current applications.
func (a *Agent) DrainDetectOnly() {
	a.mtx.RLock()
	defer a.mtx.RUnlock()
	seen := make(map[*Application]struct{}, len(a.Applications)+1)
	for _, app := range a.Applications {
		if _, ok := seen[app]; ok {
			continue
		}
		seen[app] = struct{}{}
		app.DrainDetectOnly()
	}
	// DefaultApplication is expected to be in Applications, but drain
	// it explicitly in case the invariant changes in the future.
	if a.DefaultApplication != nil {
		if _, ok := seen[a.DefaultApplication]; !ok {
			a.DefaultApplication.DrainDetectOnly()
		}
	}
}

func (a *Agent) HandleSPOE(ctx context.Context, writer *encoding.ActionWriter, message *encoding.Message) {
	timer := prometheus.NewTimer(handleSPOEDuration)
	defer timer.ObserveDuration()

	const (
		messageCorazaRequest  = "coraza-req"
		messageCorazaResponse = "coraza-res"
	)

	var messageHandler func(*Application, context.Context, *encoding.ActionWriter, *encoding.Message) error
	var isResponsePhase bool
	switch name := string(message.NameBytes()); name {
	case messageCorazaRequest:
		messageHandler = (*Application).HandleRequest
	case messageCorazaResponse:
		messageHandler = (*Application).HandleResponse
		isResponsePhase = true
	default:
		a.Logger.Debug().Str("message", name).Msg("unknown spoe message")
		return
	}

	k := encoding.AcquireKVEntry()
	defer encoding.ReleaseKVEntry(k)
	if !message.KV.Next(k) {
		a.Logger.Panic().Msg("failed reading kv entry")
		return
	}

	appName := string(k.ValueBytes())
	if !k.NameEquals("app") {
		// Without knowing the app, we cannot continue. We could fall back to a default application,
		// but all following code would have to support that as we now already read one of the kv entries.
		a.Logger.Panic().Str("expected", "app").Str("got", string(k.NameBytes())).Msg("unexpected kv entry")
		return
	}

	// On fallback, label with the default's name (cached in ReplaceApplications)
	// to bound cardinality even when HAProxy sends unbounded values (e.g.
	// hdr(host)).
	a.mtx.RLock()
	app := a.Applications[appName]
	defaultApp := a.DefaultApplication
	metricApp := appName
	if app == nil && defaultApp != nil {
		app = defaultApp
		metricApp = a.defaultApplicationName
		a.Logger.Debug().Str("app", appName).Msg("app not found, using default app")
	}
	a.mtx.RUnlock()
	if app == nil {
		a.Logger.Panic().Str("app", appName).Msg("app not found")
		return
	}

	// Verdict is final on response phase, or on request phase when
	// ResponseCheck is off. Keeps coraza_actions_total at one increment
	// per request rather than two.
	isFinalPhase := isResponsePhase || !app.ResponseCheck

	err := messageHandler(app, ctx, writer, message)
	if err == nil {
		if isFinalPhase {
			actionsTotal.WithLabelValues("allow", metricApp).Inc()
		}
		return
	}

	var interruption ErrInterrupted
	if err != nil && errors.As(err, &interruption) {
		// Interruption ends the transaction (no response phase), so it is
		// always the final verdict.
		actionsTotal.WithLabelValues(interruption.Interruption.Action, metricApp).Inc()
		_ = writer.SetInt64(encoding.VarScopeTransaction, "status", int64(interruption.Interruption.Status))
		_ = writer.SetString(encoding.VarScopeTransaction, "action", interruption.Interruption.Action)
		_ = writer.SetString(encoding.VarScopeTransaction, "data", interruption.Interruption.Data)
		_ = writer.SetInt64(encoding.VarScopeTransaction, "ruleid", int64(interruption.Interruption.RuleID))

		a.Logger.Debug().Err(err).Msg("sending interruption")
		return
	}

	// If the error is not an ErrInterrupted, we panic to let the spop stream fail.
	a.Logger.Panic().Err(err).Msg("Error handling request")
}
