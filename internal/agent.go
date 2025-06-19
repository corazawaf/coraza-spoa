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

	mtx sync.RWMutex
}

func (a *Agent) Serve(l net.Listener) error {
	agent := spop.Agent{
		Handler:     a,
		BaseContext: a.Context,
	}

	return agent.Serve(l)
}

func (a *Agent) ReplaceApplications(newApps map[string]*Application) {
	a.mtx.Lock()
	a.Applications = newApps
	a.mtx.Unlock()
}

func (a *Agent) HandleSPOE(ctx context.Context, writer *encoding.ActionWriter, message *encoding.Message) {
	timer := prometheus.NewTimer(handleSPOEDuration)
	defer timer.ObserveDuration()

	const (
		messageCorazaRequest  = "coraza-req"
		messageCorazaResponse = "coraza-res"
	)

	var messageHandler func(*Application, context.Context, *encoding.ActionWriter, *encoding.Message) error
	switch name := string(message.NameBytes()); name {
	case messageCorazaRequest:
		messageHandler = (*Application).HandleRequest
	case messageCorazaResponse:
		messageHandler = (*Application).HandleResponse
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

	a.mtx.RLock()
	app := a.Applications[appName]
	a.mtx.RUnlock()
	if app == nil && a.DefaultApplication != nil {
		// If we cannot resolve the app but the default app is configured,
		// we use the latter to process the request.
		app = a.DefaultApplication
		a.Logger.Debug().Str("app", appName).Msg("app not found, using default app")
	}
	if app == nil {
		// If we cannot resolve the app, we fail as this is an invalid configuration.
		a.Logger.Panic().Str("app", appName).Msg("app not found")
		return
	}

	err := messageHandler(app, ctx, writer, message)
	if err == nil {
		return
	}

	var interruption ErrInterrupted
	if err != nil && errors.As(err, &interruption) {
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
