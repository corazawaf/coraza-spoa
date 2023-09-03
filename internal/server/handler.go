package server

import (
	"errors"
	"fmt"

	"github.com/negasus/haproxy-spoe-go/action"
	"github.com/negasus/haproxy-spoe-go/message"
	"github.com/negasus/haproxy-spoe-go/request"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	messageCorazaRequest  = "coraza-req"
	messageCorazaResponse = "coraza-res"
)

type handler struct {
}

func (h handler) Handler(req *request.Request) {
	err := h.handler(req)
	interruption := &ErrInterrupted{}
	if err != nil && errors.As(err, interruption) {
		req.Actions.SetVar(action.ScopeTransaction, "status", interruption.Interruption.Status)
		req.Actions.SetVar(action.ScopeTransaction, "action", interruption.Interruption.Action)
		req.Actions.SetVar(action.ScopeTransaction, "data", interruption.Interruption.Data)
		req.Actions.SetVar(action.ScopeTransaction, "ruleid", interruption.Interruption.RuleID)
		if zerolog.GlobalLevel() == zerolog.DebugLevel {
			log.Debug().Err(err).Msg("Sending interruption")
		}
	} else if err != nil {
		log.Error().Err(err).Msg("Error handling request")
	}
}

func (h handler) handler(req *request.Request) error {

	var msg *message.Message
	var err error
	msg, err = req.Messages.GetByName(messageCorazaRequest)
	isRequest := true
	if err != nil {
		msg, err = req.Messages.GetByName(messageCorazaResponse)
		if err != nil {
			return errors.New("SPOE message not found")
		}
		isRequest = false
	}

	app, ok := msg.KV.Get("app")
	if !ok {
		return errors.New("App argument not received")
	}
	apps := getApps()
	// TODO in the future we should avoid using apps.Get and use applicationRequest or response
	a := apps.Get(app.(string))
	if a == nil {
		return fmt.Errorf("app %q not found", app.(string))
	}
	if isRequest {

		req := requestPool.Get().(*applicationRequest)
		defer requestPool.Put(req)

		if err := unmarshalMessage(msg, req); err != nil {
			return err
		}
		err = a.HandleRequest(req)
	} else {
		res := responsePool.Get().(*applicationResponse)
		defer responsePool.Put(req)

		if err := unmarshalMessage(msg, res); err != nil {
			return err
		}
		err = a.HandleResponse(res)
	}
	return err
}
