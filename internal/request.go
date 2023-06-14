// Copyright The OWASP Coraza contributors
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"fmt"
	"net"
	"time"

	"github.com/corazawaf/coraza/v3/types"
	spoe "github.com/criteo/haproxy-spoe-go"
	"go.uber.org/zap"
)

type request struct {
	app     string
	id      string
	srcIp   net.IP
	srcPort int
	dstIp   net.IP
	dstPort int
	method  string
	path    string
	query   string
	version string
	headers string
	body    []byte
}

func NewRequest(msg message) (*request, error) {
	req := request{}
	var err error

	req.app, err = msg.App()
	if err != nil {
		return nil, err
	}

	req.id, err = msg.Id()
	if err != nil {
		return nil, err
	}

	req.srcIp, err = msg.SrcIp()
	if err != nil {
		return nil, err
	}

	req.srcPort, err = msg.SrcPort()
	if err != nil {
		return nil, err
	}

	req.dstIp, err = msg.DstIp()
	if err != nil {
		return nil, err
	}

	req.dstPort, err = msg.DstPort()
	if err != nil {
		return nil, err
	}

	req.method, err = msg.Method()
	if err != nil {
		return nil, err
	}

	req.path, err = msg.Path()
	if err != nil {
		fmt.Println(err.Error())
		req.path = "/"
	}

	req.query, err = msg.Query()
	if err != nil {
		fmt.Println(err.Error())
	}

	req.version, err = msg.Version()
	if err != nil {
		fmt.Println(err.Error())
		req.version = "1.1"
	}

	req.headers, err = msg.Headers()
	if err != nil {
		fmt.Println(err.Error())
	}

	req.body, err = msg.Body()

	return &req, nil
}

func (s *SPOA) processRequest(spoeMsg spoe.Message) ([]spoe.Action, error) {
	var (
		err error
		req *request
		app *application
		tx  types.Transaction
	)

	defer func() {
		if tx == nil || app == nil {
			return
		}
		if tx.IsInterrupted() {
			tx.ProcessLogging()
			if err := tx.Close(); err != nil {
				app.logger.Error("failed to close transaction", zap.String("transaction_id", tx.ID()), zap.String("error", err.Error()))
			}
		} else {
			if app.cfg.NoResponseCheck {
				return
			}
			err := app.cache.SetWithExpire(tx.ID(), tx, time.Millisecond*time.Duration(app.cfg.TransactionTTLMilliseconds))
			if err != nil {
				app.logger.Error(fmt.Sprintf("failed to cache transaction: %s", err.Error()))
			}
		}
	}()

	msg := NewMessage(spoeMsg)
	req, err = NewRequest(msg)
	if err != nil {
		return nil, err
	}

	app, err = s.getApplication(req.app)
	if err != nil {
		return nil, err
	}

	tx = app.waf.NewTransactionWithID(req.id)

	headers, err := s.readHeaders(req.headers)
	if err != nil {
		return nil, err
	}
	for key, values := range headers {
		for _, v := range values {
			tx.AddRequestHeader(key, v)
		}
	}

	it, _, err := tx.WriteRequestBody(req.body)
	if err != nil {
		return nil, err
	}
	if it != nil {
		return s.processInterruption(it, hit), nil
	}

	tx.ProcessConnection(string(req.srcIp), req.srcPort, string(req.dstIp), req.dstPort)
	tx.ProcessURI(req.path+"?"+req.query, req.method, "HTTP/"+req.version)

	it = tx.ProcessRequestHeaders()
	if it != nil {
		return s.processInterruption(it, hit), nil
	}

	it, err = tx.ProcessRequestBody()
	if err != nil {
		return nil, err
	}
	if it != nil {
		return s.processInterruption(it, hit), nil
	}

	return s.message(miss), nil
}
