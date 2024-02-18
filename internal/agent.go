package internal

import (
	"context"
	"errors"
	"net"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
	"github.com/dropmorepackets/haproxy-go/spop"
	"github.com/rs/zerolog"
)

type Agent struct {
	Context      context.Context
	Applications map[string]*Application

	logger *zerolog.Logger
}

func (a *Agent) Serve(l net.Listener) error {
	agent := spop.Agent{
		Handler:     a,
		BaseContext: a.Context,
	}

	return agent.Serve(l)
}

func (a *Agent) HandleSPOE(ctx context.Context, writer *encoding.ActionWriter, message *encoding.Message) {
	const (
		messageCorazaRequest  = "coraza-req"
		messageCorazaResponse = "coraza-res"
	)

	var messageHandler func(*Application, context.Context, *encoding.Message) error
	switch name := string(message.NameBytes()); name {
	case messageCorazaRequest:
		messageHandler = (*Application).HandleRequest
	case messageCorazaResponse:
		messageHandler = (*Application).HandleResponse
	default:
		a.logger.Debug().Str("message", name).Msg("unknown spoe message")
		return
	}

	k := encoding.AcquireKVEntry()
	defer encoding.ReleaseKVEntry(k)
	if !message.KV.Next(k) {
		a.logger.Panic().Msg("failed reading kv entry")
		return
	}

	appName := string(k.ValueBytes())
	if !k.NameEquals("app") {
		// Without knowing the app, we cannot continue. We could fall back to a default application,
		// but all following code would have to support that as we now already read one of the kv entries.
		a.logger.Panic().Str("expected", "app").Str("got", appName).Msg("unexpected kv entry")
		return
	}

	app := a.Applications[appName]
	if app == nil {
		// If we cannot resolve the app, we fail as this is an invalid configuration.
		a.logger.Panic().Str("app", appName).Msg("app not found")
		return
	}

	err := messageHandler(app, ctx, message)
	if err == nil {
		return
	}

	var interruption ErrInterrupted
	if err != nil && errors.As(err, &interruption) {
		_ = writer.SetInt64(encoding.VarScopeTransaction, "status", int64(interruption.Interruption.Status))
		_ = writer.SetString(encoding.VarScopeTransaction, "action", interruption.Interruption.Action)
		_ = writer.SetString(encoding.VarScopeTransaction, "data", interruption.Interruption.Data)
		_ = writer.SetInt64(encoding.VarScopeTransaction, "ruleid", int64(interruption.Interruption.RuleID))

		a.logger.Debug().Err(err).Msg("sending interruption")
		return
	}

	// If the error is not an ErrInterrupted, we panic to let the spop stream fail.
	a.logger.Panic().Err(err).Msg("Error handling request")
}
